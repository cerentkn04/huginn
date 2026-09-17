package core

import (
	"context"
	"fmt"

	compute "cloud.google.com/go/compute/apiv1"
	computepb "cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/protobuf/proto"
)
const dockerInstallScript = `#!/bin/bash
apt-get update
apt-get install -y ca-certificates curl
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc
chmod a+r /etc/apt/keyrings/docker.asc
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
apt-get update
apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
systemctl enable docker
systemctl start docker
touch /tmp/huginn-provision-done
`
func CreateHost(ctx context.Context, projectID, zone, name string) (*computepb.Instance, error) {
	instancesClient, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcp: creating instances client: %w", err)
	}
	defer instancesClient.Close()

	req := &computepb.InsertInstanceRequest{
		Project: projectID,
		Zone:    zone,
		InstanceResource: &computepb.Instance{
			Name:        proto.String(name),
			MachineType: proto.String(fmt.Sprintf("zones/%s/machineTypes/e2-small", zone)),
			Tags: &computepb.Tags{
				Items: []string{"huginn-managed"},
			},
			Disks: []*computepb.AttachedDisk{
				{
					InitializeParams: &computepb.AttachedDiskInitializeParams{
						DiskSizeGb:  proto.Int64(20),
						SourceImage: proto.String("projects/debian-cloud/global/images/family/debian-12"),
					},
					AutoDelete: proto.Bool(true),
					Boot:       proto.Bool(true),
					Type:       proto.String(computepb.AttachedDisk_PERSISTENT.String()),
				},
			},
			NetworkInterfaces: []*computepb.NetworkInterface{
				{
					Name: proto.String("global/networks/default"),
					AccessConfigs: []*computepb.AccessConfig{
						{
							Type:        proto.String(computepb.AccessConfig_ONE_TO_ONE_NAT.String()),
							Name:        proto.String("External NAT"),
							NetworkTier: proto.String(computepb.AccessConfig_PREMIUM.String()),
						},
					},
				},
			},
			Metadata: &computepb.Metadata{
				Items: []*computepb.Items{
					{
						Key:   proto.String("startup-script"),
						Value: proto.String(dockerInstallScript),
					},
				},
			},
		},
	}

	op, err := instancesClient.Insert(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("gcp: inserting instance %s: %w", name, err)
	}
	if err := op.Wait(ctx); err != nil {
		return nil, fmt.Errorf("gcp: waiting for instance %s to be created: %w", name, err)
	}

	getReq := &computepb.GetInstanceRequest{Project: projectID, Zone: zone, Instance: name}
	inst, err := instancesClient.Get(ctx, getReq)
	if err != nil {
		return nil, fmt.Errorf("gcp: fetching created instance %s: %w", name, err)
	}
	return inst, nil
}

// DeleteHost tears down a GCP VM by name. Blocks until deletion completes.
func DeleteHost(ctx context.Context, projectID, zone, name string) error {
	instancesClient, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return fmt.Errorf("gcp: creating instances client: %w", err)
	}
	defer instancesClient.Close()

	req := &computepb.DeleteInstanceRequest{Project: projectID, Zone: zone, Instance: name}
	op, err := instancesClient.Delete(ctx, req)
	if err != nil {
		return fmt.Errorf("gcp: deleting instance %s: %w", name, err)
	}
	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("gcp: waiting for instance %s to be deleted: %w", name, err)
	}
	return nil
}

// ExternalIP returns the instance's external IPv4 address, if it has one.
func ExternalIP(inst *computepb.Instance) string {
	for _, ni := range inst.GetNetworkInterfaces() {
		for _, ac := range ni.GetAccessConfigs() {
			if ac.GetNatIP() != "" {
				return ac.GetNatIP()
			}
		}
	}
	return ""
}
