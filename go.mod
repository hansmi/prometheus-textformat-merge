module github.com/hansmi/prometheus-textformat-merge

go 1.21

// Exclude dependency on vulnerable github.com/gogo/protobuf version.
//
// https://github.com/prometheus/common/issues/315#issuecomment-1090485131
exclude github.com/gogo/protobuf v1.1.1

require (
	github.com/google/go-cmp v0.7.0
	github.com/google/renameio/v2 v2.0.0
	github.com/prometheus/client_model v0.6.1
	github.com/prometheus/common v0.63.0
	golang.org/x/sync v0.11.0
	google.golang.org/protobuf v1.36.5
)

require github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
