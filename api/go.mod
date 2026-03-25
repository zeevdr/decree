module github.com/zeevdr/central-config-service/api

go 1.24.0

require (
	google.golang.org/grpc v1.72.2
	google.golang.org/protobuf v1.36.11
)

require (
	go.opentelemetry.io/otel v1.35.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.35.0 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a // indirect
)

replace github.com/zeevdr/central-config-service/api => ./api
