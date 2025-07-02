module github.com/flext/flexcore/plugins/real-data-processor

go 1.24

toolchain go1.24.4

require (
	github.com/flext/flexcore v0.0.0
	github.com/hashicorp/go-hclog v1.6.3
	github.com/hashicorp/go-plugin v1.6.3
)

replace github.com/flext/flexcore => ../../

require (
	github.com/fatih/color v1.13.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/hashicorp/yamux v0.1.1 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/oklog/run v1.0.0 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231106174013-bbf56f31fb17 // indirect
	google.golang.org/grpc v1.61.1 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)
