package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/docker/docker/client"
	"huginn/internal/core"
)

func main() {
	ctx := context.Background()
	project := "project-79591f24-2c6a-498e-8b3"
	zone := "us-central1-a"
	name := "huginn-host-2"
	caCertPath := "/home/cerentkn04/hugin/certs/ca.pem"
	caKeyPath := "/home/cerentkn04/hugin/certs/ca-key.pem"
	clientCertPath := "/home/cerentkn04/hugin/certs/client-cert.pem"
	clientKeyPath := "/home/cerentkn04/hugin/certs/client-key.pem"

	log.Println("creating second host...")
	inst, err := core.CreateHost(ctx, project, zone, name)
	if err != nil {
		log.Fatalf("create failed: %v", err)
	}
	ip := core.InternalIP(inst)
	fmt.Printf("created %s (internal IP: %s)\n", name, ip)
	fmt.Println("waiting for host to be fully ready...")
	for i := 0; i < 30; i++ {
		time.Sleep(10 * time.Second)
		cmd := exec.Command("gcloud", "compute", "ssh", name, "--zone="+zone, "--command=cat /tmp/huginn-provision-done")
		if out, err := cmd.CombinedOutput(); err == nil {
			fmt.Println("host ready.")
			break
		} else if i == 29 {
			log.Fatalf("host never became ready: %v\n%s", err, out)
		}
	}

	log.Println("generating server cert for this host...")
	certPEM, keyPEM, err := core.GenerateServerCert(caCertPath, caKeyPath, ip)
	if err != nil {
		log.Fatalf("cert generation failed: %v", err)
	}
	if err := core.SetupRemoteTLS(zone, name, caCertPath, certPEM, keyPEM); err != nil {
		log.Fatalf("TLS setup failed: %v", err)
	}

	remoteCli, err := client.NewClientWithOpts(
		client.WithHost(fmt.Sprintf("tcp://%s:2376", ip)),
		client.WithTLSClientConfig(caCertPath, clientCertPath, clientKeyPath),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		log.Fatalf("remote docker client failed: %v", err)
	}

	localCli, err := core.NewDockerClient()
	if err != nil {
		log.Fatalf("local docker client failed: %v", err)
	}

	hostPool := core.NewHostPool()
	hostPool.Add("gamegin", localCli)
	hostPool.Add(name, remoteCli)

	fmt.Println("\n--- testing SelectHost round-robin ---")
	for i := 0; i < 5; i++ {
		id, err := hostPool.SelectHost()
		if err != nil {
			log.Fatalf("SelectHost failed: %v", err)
		}
		fmt.Printf("pick %d: %s\n", i+1, id)
	}

	cfg, err := core.LoadConfig("/home/cerentkn04/hugin/configs/example.yaml")
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}
	cfg.InternalHost = core.DiscoverInternalHost(cfg)

	log.Println("\npulling image on second host...")
	if err := core.PullImage(ctx, remoteCli, cfg.Image); err != nil {
		log.Fatalf("pull failed: %v", err)
	}
	log.Println("starting a container on the SECOND host...")
	containerID, err := core.StartInstance(ctx, remoteCli, cfg, "huginn-test-remote", 7778)
	if err != nil {
		log.Fatalf("start failed: %v", err)
	}
	fmt.Printf("container %s running on %s\n", containerID[:12], name)

	fmt.Println("\n--- testing DiscoverAllHosts ---")
	results, err := core.DiscoverAllHosts(ctx, hostPool)
	if err != nil {
		log.Fatalf("DiscoverAllHosts failed: %v", err)
	}
	for _, dh := range results {
		fmt.Printf("host %s: %d container(s)\n", dh.HostID, len(dh.Containers))
	}

	fmt.Println("\npress Enter to clean up (stop remote container + delete host)")
	fmt.Scanln()

	if err := core.StopInstance(ctx, remoteCli, containerID); err != nil {
		log.Printf("stop failed: %v", err)
	}
	if err := core.DeleteHost(ctx, project, zone, name); err != nil {
		log.Fatalf("delete failed: %v", err)
	}
	fmt.Println("cleaned up.")
}
