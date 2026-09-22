package core

import (
	"context"
	"fmt"
		"errors"
	"google.golang.org/api/iterator"
	"google.golang.org/api/googleapi"
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
GCR_HELPER_URL=$(curl -s https://api.github.com/repos/GoogleCloudPlatform/docker-credential-gcr/releases/latest | grep browser_download_url | grep linux_amd64 | cut -d '"' -f 4)
curl -fsSL "$GCR_HELPER_URL" -o /tmp/docker-credential-gcr.tar.gz
tar -xzf /tmp/docker-credential-gcr.tar.gz -C /usr/local/bin docker-credential-gcr
chmod +x /usr/local/bin/docker-credential-gcr
/usr/local/bin/docker-credential-gcr configure-docker --registries=us-central1-docker.pkg.dev
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
			ServiceAccounts: []*computepb.ServiceAccount{
				{
					Email: proto.String("default"),
					Scopes: []string{
						"https://www.googleapis.com/auth/cloud-platform",
					},
				},
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
func HostExists(ctx context.Context, projectID, zone, name string) (bool, error) {
	instancesClient, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return false, fmt.Errorf("gcp: creating instances client: %w", err)
	}
	defer instancesClient.Close()

	req := &computepb.GetInstanceRequest{Project: projectID, Zone: zone, Instance: name}
	_, err = instancesClient.Get(ctx, req)
	if err != nil {
		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) && apiErr.Code == 404 {
			return false, nil
		}
		return false, fmt.Errorf("gcp: checking instance %s: %w", name, err)
	}
	return true, nil
}
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

func InternalIP(inst *computepb.Instance) string {
	for _, ni := range inst.GetNetworkInterfaces() {
		if ni.GetNetworkIP() != "" {
			return ni.GetNetworkIP()
		}
	}
	return ""
}
func ListManagedHosts(ctx context.Context, projectID, zone string) ([]*computepb.Instance, error) {
	instancesClient, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcp: creating instances client: %w", err)
	}
	defer instancesClient.Close()

	req := &computepb.ListInstancesRequest{
		Project: projectID,
		Zone:    zone,
	}

	var results []*computepb.Instance
	it := instancesClient.List(ctx, req)
	for {
		inst, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("gcp: listing instances: %w", err)
		}
		tags := inst.GetTags().GetItems()
		for _, tag := range tags {
			if tag == "huginn-managed" {
				results = append(results, inst)
				break
			}
		}
	}
	return results, nil
}
