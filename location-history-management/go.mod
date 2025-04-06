module github.com/mmilosevicgd/location-history-management

go 1.24.2

replace github.com/mmilosevicgd/location-tracking/db => ../internal/db

replace github.com/mmilosevicgd/location-tracking/location-history-management/proto => ./proto

replace github.com/mmilosevicgd/location-tracking/model => ../internal/model

replace github.com/mmilosevicgd/location-tracking/validation => ../internal/validation

require (
	github.com/go-playground/validator/v10 v10.25.0
	github.com/mmilosevicgd/location-tracking/db v0.0.0-00010101000000-000000000000
	github.com/mmilosevicgd/location-tracking/location-history-management/proto v0.0.0-00010101000000-000000000000
	github.com/mmilosevicgd/location-tracking/model v0.0.0-00010101000000-000000000000
	github.com/mmilosevicgd/location-tracking/validation v0.0.0-00010101000000-000000000000
	go.mongodb.org/mongo-driver v1.17.2
	google.golang.org/grpc v1.71.0
	google.golang.org/protobuf v1.36.5
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.62.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	go.opentelemetry.io/otel v1.35.0 // indirect
)

require (
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/prometheus/client_golang v1.21.1
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	go.mongodb.org/mongo-driver/v2 v2.0.0 // indirect
	golang.org/x/crypto v0.35.0 // indirect
	golang.org/x/net v0.36.0 // indirect
	golang.org/x/sync v0.11.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250303144028-a0af3efb3deb // indirect
)
