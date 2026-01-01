module github.com/hansmi/prometheus-textformat-merge

go 1.24.0

toolchain go1.24.1

// Exclude dependency on vulnerable github.com/gogo/protobuf version.
//
// https://github.com/prometheus/common/issues/315#issuecomment-1090485131
exclude github.com/gogo/protobuf v1.1.1

require (
	github.com/google/go-cmp v0.7.0
	github.com/google/renameio/v2 v2.0.1
	github.com/prometheus/client_model v0.6.2
	github.com/prometheus/common v0.67.4
	golang.org/x/sync v0.19.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/kr/pretty v0.3.1 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
)
