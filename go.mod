module github.com/roadrunner-server/cache/v2

go 1.18

require (
	github.com/roadrunner-server/api/v2 v2.18.0
	github.com/roadrunner-server/endure v1.3.0
	github.com/roadrunner-server/errors v1.1.2
	github.com/roadrunner-server/sdk/v2 v2.17.3
	github.com/stretchr/testify v1.8.0
	go.buf.build/protocolbuffers/go/roadrunner-server/api v1.2.6
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.33.0
	go.opentelemetry.io/contrib/propagators/jaeger v1.8.0
	go.opentelemetry.io/otel v1.8.0
	go.opentelemetry.io/otel/trace v1.8.0
	go.uber.org/zap v1.21.0
	google.golang.org/protobuf v1.28.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/roadrunner-server/tcplisten v1.1.2 // indirect
	go.opentelemetry.io/otel/metric v0.31.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
