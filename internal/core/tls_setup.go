package core

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"time"
)
func parsePrivateKey(der []byte) (*rsa.PrivateKey, error) {
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("not a valid PKCS1 or PKCS8 RSA key: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("CA key is not an RSA key")
	}
	return rsaKey, nil
}
func GenerateServerCert(caCertPath, caKeyPath, ip string) (certPEM, keyPEM []byte, err error) {
	caCertBytes, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, nil, fmt.Errorf("reading CA cert: %w", err)
	}
	caKeyBytes, err := os.ReadFile(caKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("reading CA key: %w", err)
	}

	caCertBlock, _ := pem.Decode(caCertBytes)
	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing CA cert: %w", err)
	}
	caKeyBlock, _ := pem.Decode(caKeyBytes)
	caKey, err := parsePrivateKey(caKeyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing CA key: %w", err)
	}

	serverKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, fmt.Errorf("generating server key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("generating serial: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: ip},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP(ip)},
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("signing server cert: %w", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverKey)})
	return certPEM, keyPEM, nil
}

func SetupRemoteTLS(zone, instanceName string, caCertPath string, serverCertPEM, serverKeyPEM []byte) error {
	tmpCert, err := os.CreateTemp("", "server-cert-*.pem")
	if err != nil {
		return fmt.Errorf("creating temp cert file: %w", err)
	}
	defer os.Remove(tmpCert.Name())
	if _, err := tmpCert.Write(serverCertPEM); err != nil {
		return fmt.Errorf("writing temp cert: %w", err)
	}
	tmpCert.Close()

	tmpKey, err := os.CreateTemp("", "server-key-*.pem")
	if err != nil {
		return fmt.Errorf("creating temp key file: %w", err)
	}
	defer os.Remove(tmpKey.Name())
	if _, err := tmpKey.Write(serverKeyPEM); err != nil {
		return fmt.Errorf("writing temp key: %w", err)
	}
	tmpKey.Close()
scpArgs := []string{
    caCertPath,
    tmpCert.Name(),
    tmpKey.Name(),
    fmt.Sprintf("%s:/tmp/", instanceName),
    "--zone=" + zone,
}

	if out, err := exec.Command("gcloud", append([]string{"compute", "scp"}, scpArgs...)...).CombinedOutput(); err != nil {
		return fmt.Errorf("scp to %s failed: %w\n%s", instanceName, err, out)
	}

	remoteScript := fmt.Sprintf(`
sudo mkdir -p /etc/docker/certs
sudo mv /tmp/%s /etc/docker/certs/ca.pem
sudo mv /tmp/%s /etc/docker/certs/server-cert.pem
sudo mv /tmp/%s /etc/docker/certs/server-key.pem
sudo chmod 600 /etc/docker/certs/server-key.pem
sudo mkdir -p /etc/systemd/system/docker.service.d
sudo tee /etc/systemd/system/docker.service.d/tls.conf > /dev/null <<'CONFEOF'
[Service]
ExecStart=
ExecStart=/usr/bin/dockerd -H unix:///var/run/docker.sock -H tcp://0.0.0.0:2376 --tlsverify --tlscacert=/etc/docker/certs/ca.pem --tlscert=/etc/docker/certs/server-cert.pem --tlskey=/etc/docker/certs/server-key.pem
CONFEOF
sudo systemctl daemon-reload
sudo systemctl restart docker
`, baseName(caCertPath), tmpCert.Name()[len("/tmp/"):], tmpKey.Name()[len("/tmp/"):])

	sshArgs := []string{"compute", "ssh", instanceName, "--zone=" + zone, "--command=" + remoteScript}
		cmd := exec.Command("gcloud", sshArgs...)
cmd.Stdin = os.Stdin
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr

if err := cmd.Run(); err != nil {
    return fmt.Errorf("ssh config on %s failed: %w", instanceName, err)
}
return nil
}

func baseName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}
