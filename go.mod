module github.com/hansmi/prometheus-textformat-merge

go 1.15

// Exclude dependency on vulnerable github.com/gogo/protobuf version.
//
// https://github.com/prometheus/common/issues/315#issuecomment-1090485131
exclude github.com/gogo/protobuf v1.1.1

require (
	github.com/google/go-cmp v0.5.5
	github.com/google/renameio/v2 v2.0.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.34.0
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	google.golang.org/protobuf v1.28.0 // indirect
)
