package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type projectInfo struct {
	CommonInstanceMetadata struct {
		Items []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"items"`
	} `json:"commonInstanceMetadata"`
}

func promptGCPAutomatedSetup(reader *bufio.Reader, project string) {
	if !promptYesNo(reader, "Run automated GCP setup now? (enables APIs, grants IAM roles, adds an SSH key — makes real changes to your project)", false) {
		fmt.Println("Skipped — see GET_STARTED.md for the manual steps.")
		return
	}

	account, err := runGcloudCapture("auth", "list", "--filter=status:ACTIVE", "--format=value(account)")
	if err != nil || strings.TrimSpace(account) == "" {
		fmt.Println("gcloud is not installed or not authenticated. Run `gcloud auth login` and re-run `huginn init`.")
		return
	}
	fmt.Printf("Acting as: %s\n", strings.TrimSpace(account))
	acct := strings.TrimSpace(account)
	if strings.HasSuffix(acct, ".gserviceaccount.com") {
		fmt.Println("Warning: gcloud is authenticated as a service account, which usually cannot grant IAM roles.")
		fmt.Println("Run `gcloud auth login` as a project owner first (or run `huginn init` from Cloud Shell).")
		if !promptYesNo(reader, "Continue anyway?", false) {
			return
		}
	}

	if _, err := runGcloudCapture("projects", "describe", project, "--format=value(projectId)"); err != nil {
		fmt.Printf("Project %q was not found, or %s has no access to it. Check the ID and your gcloud account.\n", project, acct)
		return
	}
	if err := enableAPIs(project); err != nil {
		fmt.Fprintf(os.Stderr, "huginn: enabling APIs failed: %v\n", err)
		return
	}

	saEmail, err := resolveServiceAccount(project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "huginn: could not determine service account: %v\n", err)
		return
	}
	fmt.Printf("Service account Huginn will use: %s\n", saEmail)

	if err := grantIAMRoles(project, saEmail); err != nil {
		fmt.Fprintf(os.Stderr, "huginn: IAM setup failed: %v\n", err)
		fmt.Println("Grant the roles manually — see GET_STARTED.md.")
		return
	}

	if err := setupAutomationSSHKey(reader, project); err != nil {
		fmt.Fprintf(os.Stderr, "huginn: SSH key setup failed: %v\n", err)
		fmt.Println("Add the key manually — see GET_STARTED.md.")
		return
	}

	fmt.Println("GCP automated setup complete.")
}

func enableAPIs(project string) error {
	fmt.Println("Enabling Compute Engine and Artifact Registry APIs (no-op if already enabled)...")
	cmd := exec.Command("gcloud", "services", "enable",
		"compute.googleapis.com", "artifactregistry.googleapis.com", "--project="+project)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// resolveServiceAccount prefers the identity of the VM we're running on (the
// account Huginn will actually make API calls as). Off-GCP, it falls back to
// the project's default compute service account.
func resolveServiceAccount(project string) (string, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequest("GET",
		"http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/email", nil)
	req.Header.Set("Metadata-Flavor", "Google")
	if resp, err := client.Do(req); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			b, _ := io.ReadAll(resp.Body)
			if email := strings.TrimSpace(string(b)); email != "" {
				return email, nil
			}
		}
	}

	fmt.Println("Not running on a GCE VM — falling back to the project's default compute service account.")
	num, err := runGcloudCapture("projects", "describe", project, "--format=value(projectNumber)")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-compute@developer.gserviceaccount.com", strings.TrimSpace(num)), nil
}

func grantIAMRoles(project, saEmail string) error {
	member := "serviceAccount:" + saEmail

	for _, role := range []string{"roles/compute.instanceAdmin.v1", "roles/artifactregistry.writer"} {
		fmt.Printf("Granting %s...\n", role)
		cmd := exec.Command("gcloud", "projects", "add-iam-policy-binding", project,
			"--member="+member, "--role="+role, "--condition=None", "--quiet")
		cmd.Stdout = io.Discard // policy dump is huge and useless here
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("granting %s: %w", role, err)
		}
	}

	// Scoped to the SA itself, NOT project-wide — lets it attach itself to new
	// VMs without being able to impersonate every other SA in the project.
	fmt.Println("Granting roles/iam.serviceAccountUser (on the service account itself)...")
	cmd := exec.Command("gcloud", "iam", "service-accounts", "add-iam-policy-binding", saEmail,
		"--project="+project, "--member="+member, "--role=roles/iam.serviceAccountUser", "--quiet")
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("granting serviceAccountUser: %w", err)
	}
	return nil
}

func setupAutomationSSHKey(reader *bufio.Reader, project string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("finding home directory: %w", err)
	}
	keyPath := filepath.Join(home, ".ssh", "huginn_automation_key")
	pubKeyPath := keyPath + ".pub"

	if _, err := os.Stat(pubKeyPath); os.IsNotExist(err) {
		fmt.Println("Generating passphrase-less SSH key for automated provisioning...")
		if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
			return fmt.Errorf("creating ~/.ssh: %w", err)
		}
		cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", "", "-q")
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("generating SSH key: %w", err)
		}
	} else {
		fmt.Printf("Reusing existing key at %s\n", keyPath)
	}

	pubKeyBytes, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return fmt.Errorf("reading public key: %w", err)
	}
	pubKey := strings.TrimSpace(string(pubKeyBytes))

	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("LOGNAME")
	}
	if user == "" {
		return fmt.Errorf("could not determine current username")
	}
	newEntry := fmt.Sprintf("%s:%s", user, pubKey)

	// Read-before-write: project ssh-keys must be appended to, never overwritten.
	infoJSON, err := runGcloudCapture("compute", "project-info", "describe", "--project="+project, "--format=json")
	if err != nil {
		return fmt.Errorf("reading project metadata: %w", err)
	}
	var info projectInfo
	if err := json.Unmarshal([]byte(infoJSON), &info); err != nil {
		return fmt.Errorf("parsing project metadata: %w", err)
	}

	existing := ""
	for _, item := range info.CommonInstanceMetadata.Items {
		if item.Key == "ssh-keys" {
			existing = item.Value
			break
		}
	}

	if strings.Contains(existing, pubKey) {
		fmt.Println("Key already present in project SSH metadata — nothing to do.")
		return nil
	}

	existingCount := 0
	if strings.TrimSpace(existing) != "" {
		existingCount = len(strings.Split(strings.TrimSpace(existing), "\n"))
	}
	fmt.Printf("Project currently has %d SSH key(s). Huginn will ADD 1 key and keep all existing ones.\n", existingCount)
	if !promptYesNo(reader, "Proceed with updating project SSH metadata?", false) {
		fmt.Println("Skipped SSH key metadata update.")
		return nil
	}

	combined := newEntry
	if strings.TrimSpace(existing) != "" {
		combined = strings.TrimRight(existing, "\n") + "\n" + newEntry
	}

	tmpFile, err := os.CreateTemp("", "huginn-ssh-keys-*.txt")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(combined); err != nil {
		tmpFile.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	tmpFile.Close()

	cmd := exec.Command("gcloud", "compute", "project-info", "add-metadata",
		"--project="+project, "--metadata-from-file=ssh-keys="+tmpFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("updating metadata: %w", err)
	}
	return nil
}

func runGcloudCapture(args ...string) (string, error) {
	out, err := exec.Command("gcloud", args...).Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%v: %s", err, string(exitErr.Stderr))
		}
		return "", err
	}
	return string(out), nil
}
