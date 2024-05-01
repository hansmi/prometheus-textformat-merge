module github.com/hansmi/prometheus-textformat-merge

go 1.21

toolchain go1.21.7

// Exclude dependency on vulnerable github.com/gogo/protobuf version.
//
// https://github.com/prometheus/common/issues/315#issuecomment-1090485131
exclude github.com/gogo/protobuf v1.1.1

require (
	github.com/google/go-cmp v0.6.0
	github.com/google/renameio/v2 v2.0.0
	github.com/prometheus/client_model v0.6.1
	github.com/prometheus/common v0.53.0
	golang.org/x/sync v0.7.0
	google.golang.org/protobuf v1.34.0
)
