module huginn

go 1.23

toolchain go1.24.7

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/distribution/reference v0.5.0 // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker v24.0.9+incompatible // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/gogo/protobuf v1.2.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/net v0.0.0-00010101000000-000000000000 // indirect
)

replace golang.org/x/net => github.com/golang/net v0.30.0

replace golang.org/x/sys => github.com/golang/sys v0.28.0

replace golang.org/x/time => github.com/golang/time v0.8.0
