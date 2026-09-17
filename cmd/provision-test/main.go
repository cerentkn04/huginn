package main

import (
	"context"
	"fmt"
	"log"

	compute "cloud.google.com/go/compute/apiv1"
	computepb "cloud.google.com/go/compute/apiv1/computepb"
	"huginn/internal/core"
)

func main() {
	ctx := context.Background()
	project := "project-79591f24-2c6a-498e-8b3"
	zone := "us-central1-a"
	name := "huginn-provision-test"
	caCertPath := "/home/cerentkn04/hugin/certs/ca.pem"
	caKeyPath := "/home/cerentkn04/hugin/certs/ca-key.pem"

	instancesClient, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		log.Fatalf("client: %v", err)
	}
	defer instancesClient.Close()

	inst, err := instancesClient.Get(ctx, &computepb.GetInstanceRequest{Project: project, Zone: zone, Instance: name})
	if err != nil {
		log.Fatalf("get instance failed: %v", err)
	}
	ip := core.InternalIP(inst)
	fmt.Printf("using instance %s (internal IP: %s)\n", name, ip)

	log.Println("generating server cert for this IP...")
	certPEM, keyPEM, err := core.GenerateServerCert(caCertPath, caKeyPath, ip)
	if err != nil {
		log.Fatalf("cert generation failed: %v", err)
	}

	log.Println("pushing TLS config to host...")
	if err := core.SetupRemoteTLS(zone, name, caCertPath, certPEM, keyPEM); err != nil {
		log.Fatalf("TLS setup failed: %v", err)
	}

	fmt.Println("done — now test TLS connectivity manually.")
}
