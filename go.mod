module huginn

go 1.24

require (
	github.com/docker/docker v24.0.9+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/goccy/go-yaml v1.19.2
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/distribution/reference v0.5.0 // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/gogo/protobuf v1.2.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/morikuni/aec v1.1.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/stretchr/testify v1.12.1 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	gotest.tools/v3 v3.5.2 // indirect
)

replace golang.org/x/net => github.com/golang/net v0.30.0

replace golang.org/x/sys => github.com/golang/sys v0.28.0

replace golang.org/x/time => github.com/golang/time v0.8.0
