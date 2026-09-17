package core

import "context"

type Host struct {
	Name       string
	ExternalIP string
	InternalIP string
}

type HostProvisioner interface {
	CreateHost(ctx context.Context, name string) (*Host, error)
	DeleteHost(ctx context.Context, name string) error
}
