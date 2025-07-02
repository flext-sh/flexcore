module github.com/flext/flexcore

go 1.24

toolchain go1.24.4

require (
	github.com/flext/flexcore/infrastructure/cqrs v0.0.0-00010101000000-000000000000
	github.com/flext/flexcore/infrastructure/eventsourcing v0.0.0-00010101000000-000000000000
	github.com/flext/flexcore/infrastructure/plugins v0.0.0-00010101000000-000000000000
	github.com/flext/flexcore/infrastructure/windmill v0.0.0-00010101000000-000000000000
	github.com/fsnotify/fsnotify v1.9.0
	github.com/gin-gonic/gin v1.9.1
	github.com/go-playground/validator/v10 v10.26.0
	github.com/go-redis/redis/v8 v8.11.5
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/hashicorp/go-hclog v0.14.1
	github.com/hashicorp/go-plugin v1.6.3
	github.com/lib/pq v1.10.9
	github.com/mattn/go-sqlite3 v1.14.22
	github.com/prometheus/client_golang v1.22.0
	github.com/redis/go-redis/v9 v9.11.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/spf13/viper v1.20.1
	github.com/stretchr/testify v1.10.0
	go.etcd.io/etcd/client/v3 v3.6.1
	go.opentelemetry.io/otel v1.37.0
	go.opentelemetry.io/otel/trace v1.37.0
	golang.org/x/net v0.38.0
	google.golang.org/grpc v1.71.1
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bytedance/sonic v1.10.0-rc3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20230717121745-296ad89f973d // indirect
	github.com/chenzhuoyu/iasm v0.9.0 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/fatih/color v1.7.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.3.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.26.3 // indirect
	github.com/hashicorp/yamux v0.1.1 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.5 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oklog/run v1.0.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.62.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/sagikazarmark/locafero v0.9.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.14.0 // indirect
	github.com/spf13/cast v1.9.2 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.11 // indirect
	go.etcd.io/etcd/api/v3 v3.6.1 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.6.1 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/arch v0.4.0 // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250303144028-a0af3efb3deb // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250303144028-a0af3efb3deb // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/flext/flexcore/infrastructure/cqrs => ././infrastructure/cqrs
	github.com/flext/flexcore/infrastructure/eventsourcing => ././infrastructure/eventsourcing
	github.com/flext/flexcore/infrastructure/plugins => ././infrastructure/plugins
	github.com/flext/flexcore/infrastructure/windmill => ././infrastructure/windmill
	github.com/flext/flexcore/plugins => ././plugins
	github.com/flext/flexcore/plugins/api-loader => ././plugins/api-loader
	github.com/flext/flexcore/plugins/data-processor => ././plugins/data-processor
	github.com/flext/flexcore/plugins/data-transformer => ././plugins/data-transformer
	github.com/flext/flexcore/plugins/json-processor => ././plugins/json-processor
	github.com/flext/flexcore/plugins/json-transformer => ././plugins/json-transformer
	github.com/flext/flexcore/plugins/plugins => ././plugins/plugins
	github.com/flext/flexcore/plugins/postgres-extractor => ././plugins/postgres-extractor
	github.com/flext/flexcore/plugins/postgres-processor => ././plugins/postgres-processor
	github.com/flext/flexcore/plugins/real-data-processor => ././plugins/real-data-processor
	github.com/flext/flexcore/plugins/simple-processor => ././plugins/simple-processor
	github.com/flext/flexcore/plugins/simple-processor/vendor => ././plugins/simple-processor/vendor
	github.com/flext/flexcore/plugins/simple-processor/vendor/github.com => ././plugins/simple-processor/vendor/github.com
	github.com/flext/flexcore/plugins/simple-processor/vendor/github.com/fatih => ././plugins/simple-processor/vendor/github.com/fatih
	github.com/flext/flexcore/plugins/simple-processor/vendor/github.com/fatih/color => ././plugins/simple-processor/vendor/github.com/fatih/color
	github.com/flext/flexcore/plugins/simple-processor/vendor/github.com/golang => ././plugins/simple-processor/vendor/github.com/golang
	github.com/flext/flexcore/plugins/simple-processor/vendor/github.com/golang/protobuf => ././plugins/simple-processor/vendor/github.com/golang/protobuf
	github.com/flext/flexcore/plugins/simple-processor/vendor/github.com/golang/protobuf/ptypes => ././plugins/simple-processor/vendor/github.com/golang/protobuf/ptypes
	github.com/flext/flexcore/plugins/simple-processor/vendor/github.com/golang/protobuf/ptypes/empty => ././plugins/simple-processor/vendor/github.com/golang/protobuf/ptypes/empty
	github.com/flext/flexcore/plugins/simple-processor/vendor/github.com/hashicorp => ././plugins/simple-processor/vendor/github.com/hashicorp
	github.com/flext/flexcore/plugins/simple-processor/vendor/github.com/hashicorp/go-hclog => ././plugins/simple-processor/vendor/github.com/hashicorp/go-hclog
	github.com/flext/flexcore/plugins/simple-processor/vendor/github.com/hashicorp/go-plugin => ././plugins/simple-processor/vendor/github.com/hashicorp/go-plugin
	github.com/flext/flexcore/plugins/simple-processor/vendor/github.com/hashicorp/go-plugin/internal => ././plugins/simple-processor/vendor/github.com/hashicorp/go-plugin/internal
	github.com/flext/flexcore/plugins/simple-processor/vendor/github.com/hashicorp/go-plugin/internal/cmdrunner => ././plugins/simple-processor/vendor/github.com/hashicorp/go-plugin/internal/cmdrunner
	github.com/flext/flexcore/plugins/simple-processor/vendor/github.com/hashicorp/go-plugin/internal/grpcmux => ././plugins/simple-processor/vendor/github.com/hashicorp/go-plugin/internal/grpcmux
	github.com/flext/flexcore/plugins/simple-processor/vendor/github.com/hashicorp/go-plugin/internal/plugin => ././plugins/simple-processor/vendor/github.com/hashicorp/go-plugin/internal/plugin
	github.com/flext/flexcore/plugins/simple-processor/vendor/github.com/hashicorp/go-plugin/runner => ././plugins/simple-processor/vendor/github.com/hashicorp/go-plugin/runner
	github.com/flext/flexcore/plugins/simple-processor/vendor/github.com/hashicorp/yamux => ././plugins/simple-processor/vendor/github.com/hashicorp/yamux
	github.com/flext/flexcore/plugins/simple-processor/vendor/github.com/mattn => ././plugins/simple-processor/vendor/github.com/mattn
	github.com/flext/flexcore/plugins/simple-processor/vendor/github.com/mattn/go-colorable => ././plugins/simple-processor/vendor/github.com/mattn/go-colorable
	github.com/flext/flexcore/plugins/simple-processor/vendor/github.com/mattn/go-isatty => ././plugins/simple-processor/vendor/github.com/mattn/go-isatty
	github.com/flext/flexcore/plugins/simple-processor/vendor/github.com/oklog => ././plugins/simple-processor/vendor/github.com/oklog
	github.com/flext/flexcore/plugins/simple-processor/vendor/github.com/oklog/run => ././plugins/simple-processor/vendor/github.com/oklog/run
	github.com/flext/flexcore/plugins/simple-processor/vendor/golang.org => ././plugins/simple-processor/vendor/golang.org
	github.com/flext/flexcore/plugins/simple-processor/vendor/golang.org/x => ././plugins/simple-processor/vendor/golang.org/x
	github.com/flext/flexcore/plugins/simple-processor/vendor/golang.org/x/net => ././plugins/simple-processor/vendor/golang.org/x/net
	github.com/flext/flexcore/plugins/simple-processor/vendor/golang.org/x/net/http => ././plugins/simple-processor/vendor/golang.org/x/net/http
	github.com/flext/flexcore/plugins/simple-processor/vendor/golang.org/x/net/http/httpguts => ././plugins/simple-processor/vendor/golang.org/x/net/http/httpguts
	github.com/flext/flexcore/plugins/simple-processor/vendor/golang.org/x/net/http2 => ././plugins/simple-processor/vendor/golang.org/x/net/http2
	github.com/flext/flexcore/plugins/simple-processor/vendor/golang.org/x/net/http2/hpack => ././plugins/simple-processor/vendor/golang.org/x/net/http2/hpack
	github.com/flext/flexcore/plugins/simple-processor/vendor/golang.org/x/net/idna => ././plugins/simple-processor/vendor/golang.org/x/net/idna
	github.com/flext/flexcore/plugins/simple-processor/vendor/golang.org/x/net/internal => ././plugins/simple-processor/vendor/golang.org/x/net/internal
	github.com/flext/flexcore/plugins/simple-processor/vendor/golang.org/x/net/internal/httpcommon => ././plugins/simple-processor/vendor/golang.org/x/net/internal/httpcommon
	github.com/flext/flexcore/plugins/simple-processor/vendor/golang.org/x/net/internal/timeseries => ././plugins/simple-processor/vendor/golang.org/x/net/internal/timeseries
	github.com/flext/flexcore/plugins/simple-processor/vendor/golang.org/x/net/trace => ././plugins/simple-processor/vendor/golang.org/x/net/trace
	github.com/flext/flexcore/plugins/simple-processor/vendor/golang.org/x/sys => ././plugins/simple-processor/vendor/golang.org/x/sys
	github.com/flext/flexcore/plugins/simple-processor/vendor/golang.org/x/sys/unix => ././plugins/simple-processor/vendor/golang.org/x/sys/unix
	github.com/flext/flexcore/plugins/simple-processor/vendor/golang.org/x/sys/windows => ././plugins/simple-processor/vendor/golang.org/x/sys/windows
	github.com/flext/flexcore/plugins/simple-processor/vendor/golang.org/x/text => ././plugins/simple-processor/vendor/golang.org/x/text
	github.com/flext/flexcore/plugins/simple-processor/vendor/golang.org/x/text/secure => ././plugins/simple-processor/vendor/golang.org/x/text/secure
	github.com/flext/flexcore/plugins/simple-processor/vendor/golang.org/x/text/secure/bidirule => ././plugins/simple-processor/vendor/golang.org/x/text/secure/bidirule
	github.com/flext/flexcore/plugins/simple-processor/vendor/golang.org/x/text/transform => ././plugins/simple-processor/vendor/golang.org/x/text/transform
	github.com/flext/flexcore/plugins/simple-processor/vendor/golang.org/x/text/unicode => ././plugins/simple-processor/vendor/golang.org/x/text/unicode
	github.com/flext/flexcore/plugins/simple-processor/vendor/golang.org/x/text/unicode/bidi => ././plugins/simple-processor/vendor/golang.org/x/text/unicode/bidi
	github.com/flext/flexcore/plugins/simple-processor/vendor/golang.org/x/text/unicode/norm => ././plugins/simple-processor/vendor/golang.org/x/text/unicode/norm
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org => ././plugins/simple-processor/vendor/google.golang.org
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/genproto => ././plugins/simple-processor/vendor/google.golang.org/genproto
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/genproto/googleapis => ././plugins/simple-processor/vendor/google.golang.org/genproto/googleapis
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/genproto/googleapis/rpc => ././plugins/simple-processor/vendor/google.golang.org/genproto/googleapis/rpc
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/genproto/googleapis/rpc/status => ././plugins/simple-processor/vendor/google.golang.org/genproto/googleapis/rpc/status
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc => ././plugins/simple-processor/vendor/google.golang.org/grpc
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/attributes => ././plugins/simple-processor/vendor/google.golang.org/grpc/attributes
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/backoff => ././plugins/simple-processor/vendor/google.golang.org/grpc/backoff
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/balancer => ././plugins/simple-processor/vendor/google.golang.org/grpc/balancer
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/balancer/base => ././plugins/simple-processor/vendor/google.golang.org/grpc/balancer/base
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/balancer/endpointsharding => ././plugins/simple-processor/vendor/google.golang.org/grpc/balancer/endpointsharding
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/balancer/grpclb => ././plugins/simple-processor/vendor/google.golang.org/grpc/balancer/grpclb
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/balancer/grpclb/state => ././plugins/simple-processor/vendor/google.golang.org/grpc/balancer/grpclb/state
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/balancer/pickfirst => ././plugins/simple-processor/vendor/google.golang.org/grpc/balancer/pickfirst
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/balancer/pickfirst/internal => ././plugins/simple-processor/vendor/google.golang.org/grpc/balancer/pickfirst/internal
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/balancer/pickfirst/pickfirstleaf => ././plugins/simple-processor/vendor/google.golang.org/grpc/balancer/pickfirst/pickfirstleaf
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/balancer/roundrobin => ././plugins/simple-processor/vendor/google.golang.org/grpc/balancer/roundrobin
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/binarylog => ././plugins/simple-processor/vendor/google.golang.org/grpc/binarylog
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/binarylog/grpc_binarylog_v1 => ././plugins/simple-processor/vendor/google.golang.org/grpc/binarylog/grpc_binarylog_v1
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/channelz => ././plugins/simple-processor/vendor/google.golang.org/grpc/channelz
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/codes => ././plugins/simple-processor/vendor/google.golang.org/grpc/codes
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/connectivity => ././plugins/simple-processor/vendor/google.golang.org/grpc/connectivity
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/credentials => ././plugins/simple-processor/vendor/google.golang.org/grpc/credentials
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/credentials/insecure => ././plugins/simple-processor/vendor/google.golang.org/grpc/credentials/insecure
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/encoding => ././plugins/simple-processor/vendor/google.golang.org/grpc/encoding
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/encoding/proto => ././plugins/simple-processor/vendor/google.golang.org/grpc/encoding/proto
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/experimental => ././plugins/simple-processor/vendor/google.golang.org/grpc/experimental
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/experimental/stats => ././plugins/simple-processor/vendor/google.golang.org/grpc/experimental/stats
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/grpclog => ././plugins/simple-processor/vendor/google.golang.org/grpc/grpclog
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/grpclog/internal => ././plugins/simple-processor/vendor/google.golang.org/grpc/grpclog/internal
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/health => ././plugins/simple-processor/vendor/google.golang.org/grpc/health
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/health/grpc_health_v1 => ././plugins/simple-processor/vendor/google.golang.org/grpc/health/grpc_health_v1
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/backoff => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/backoff
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/balancer => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/balancer
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/balancer/gracefulswitch => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/balancer/gracefulswitch
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/balancerload => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/balancerload
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/binarylog => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/binarylog
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/buffer => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/buffer
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/channelz => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/channelz
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/credentials => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/credentials
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/envconfig => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/envconfig
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/grpclog => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/grpclog
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/grpcsync => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/grpcsync
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/grpcutil => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/grpcutil
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/idle => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/idle
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/metadata => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/metadata
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/pretty => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/pretty
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/proxyattributes => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/proxyattributes
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/resolver => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/resolver
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/resolver/delegatingresolver => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/resolver/delegatingresolver
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/resolver/dns => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/resolver/dns
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/resolver/dns/internal => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/resolver/dns/internal
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/resolver/passthrough => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/resolver/passthrough
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/resolver/unix => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/resolver/unix
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/serviceconfig => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/serviceconfig
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/stats => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/stats
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/status => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/status
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/syscall => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/syscall
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/transport => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/transport
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/internal/transport/networktype => ././plugins/simple-processor/vendor/google.golang.org/grpc/internal/transport/networktype
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/keepalive => ././plugins/simple-processor/vendor/google.golang.org/grpc/keepalive
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/mem => ././plugins/simple-processor/vendor/google.golang.org/grpc/mem
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/metadata => ././plugins/simple-processor/vendor/google.golang.org/grpc/metadata
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/peer => ././plugins/simple-processor/vendor/google.golang.org/grpc/peer
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/reflection => ././plugins/simple-processor/vendor/google.golang.org/grpc/reflection
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/reflection/grpc_reflection_v1 => ././plugins/simple-processor/vendor/google.golang.org/grpc/reflection/grpc_reflection_v1
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/reflection/grpc_reflection_v1alpha => ././plugins/simple-processor/vendor/google.golang.org/grpc/reflection/grpc_reflection_v1alpha
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/reflection/internal => ././plugins/simple-processor/vendor/google.golang.org/grpc/reflection/internal
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/resolver => ././plugins/simple-processor/vendor/google.golang.org/grpc/resolver
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/resolver/dns => ././plugins/simple-processor/vendor/google.golang.org/grpc/resolver/dns
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/serviceconfig => ././plugins/simple-processor/vendor/google.golang.org/grpc/serviceconfig
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/stats => ././plugins/simple-processor/vendor/google.golang.org/grpc/stats
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/status => ././plugins/simple-processor/vendor/google.golang.org/grpc/status
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/grpc/tap => ././plugins/simple-processor/vendor/google.golang.org/grpc/tap
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf => ././plugins/simple-processor/vendor/google.golang.org/protobuf
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/encoding => ././plugins/simple-processor/vendor/google.golang.org/protobuf/encoding
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/encoding/protojson => ././plugins/simple-processor/vendor/google.golang.org/protobuf/encoding/protojson
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/encoding/prototext => ././plugins/simple-processor/vendor/google.golang.org/protobuf/encoding/prototext
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/encoding/protowire => ././plugins/simple-processor/vendor/google.golang.org/protobuf/encoding/protowire
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/descfmt => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/descfmt
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/descopts => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/descopts
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/detrand => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/detrand
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/editiondefaults => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/editiondefaults
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/editionssupport => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/editionssupport
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/encoding => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/encoding
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/encoding/defval => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/encoding/defval
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/encoding/json => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/encoding/json
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/encoding/messageset => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/encoding/messageset
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/encoding/tag => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/encoding/tag
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/encoding/text => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/encoding/text
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/errors => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/errors
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/filedesc => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/filedesc
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/filetype => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/filetype
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/flags => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/flags
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/genid => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/genid
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/impl => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/impl
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/order => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/order
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/pragma => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/pragma
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/protolazy => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/protolazy
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/set => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/set
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/strs => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/strs
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/internal/version => ././plugins/simple-processor/vendor/google.golang.org/protobuf/internal/version
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/proto => ././plugins/simple-processor/vendor/google.golang.org/protobuf/proto
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/protoadapt => ././plugins/simple-processor/vendor/google.golang.org/protobuf/protoadapt
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/reflect => ././plugins/simple-processor/vendor/google.golang.org/protobuf/reflect
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/reflect/protodesc => ././plugins/simple-processor/vendor/google.golang.org/protobuf/reflect/protodesc
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/reflect/protoreflect => ././plugins/simple-processor/vendor/google.golang.org/protobuf/reflect/protoreflect
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/reflect/protoregistry => ././plugins/simple-processor/vendor/google.golang.org/protobuf/reflect/protoregistry
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/runtime => ././plugins/simple-processor/vendor/google.golang.org/protobuf/runtime
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/runtime/protoiface => ././plugins/simple-processor/vendor/google.golang.org/protobuf/runtime/protoiface
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/runtime/protoimpl => ././plugins/simple-processor/vendor/google.golang.org/protobuf/runtime/protoimpl
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/types => ././plugins/simple-processor/vendor/google.golang.org/protobuf/types
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/types/descriptorpb => ././plugins/simple-processor/vendor/google.golang.org/protobuf/types/descriptorpb
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/types/gofeaturespb => ././plugins/simple-processor/vendor/google.golang.org/protobuf/types/gofeaturespb
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/types/known => ././plugins/simple-processor/vendor/google.golang.org/protobuf/types/known
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/types/known/anypb => ././plugins/simple-processor/vendor/google.golang.org/protobuf/types/known/anypb
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/types/known/durationpb => ././plugins/simple-processor/vendor/google.golang.org/protobuf/types/known/durationpb
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/types/known/emptypb => ././plugins/simple-processor/vendor/google.golang.org/protobuf/types/known/emptypb
	github.com/flext/flexcore/plugins/simple-processor/vendor/google.golang.org/protobuf/types/known/timestamppb => ././plugins/simple-processor/vendor/google.golang.org/protobuf/types/known/timestamppb
)
