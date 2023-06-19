module github.com/hansmi/prometheus-textformat-merge

go 1.19

// Exclude dependency on vulnerable github.com/gogo/protobuf version.
//
// https://github.com/prometheus/common/issues/315#issuecomment-1090485131
exclude github.com/gogo/protobuf v1.1.1

require (
	github.com/google/go-cmp v0.5.9
	github.com/google/renameio/v2 v2.0.0
	github.com/prometheus/client_model v0.3.0
	github.com/prometheus/common v0.42.0
	golang.org/x/sync v0.3.0
)

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)
