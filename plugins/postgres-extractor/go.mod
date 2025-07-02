module github.com/flext/flexcore/plugins/postgres-extractor

go 1.24

toolchain go1.24.4

require (
	github.com/flext/flexcore v0.0.0
	github.com/hashicorp/go-hclog v1.6.2
	github.com/hashicorp/go-plugin v1.6.3
	github.com/lib/pq v1.10.9
)

replace github.com/flext/flexcore => ../..

require (
	github.com/fatih/color v1.13.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/hashicorp/yamux v0.1.1 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/oklog/run v1.0.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250303144028-a0af3efb3deb // indirect
	google.golang.org/grpc v1.71.1 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)
