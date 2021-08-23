# Utility to merge Prometheus textformat files

[![Latest release](https://img.shields.io/github/v/release/hansmi/prometheus-textformat-merge)][releases]
[![Release workflow](https://github.com/hansmi/prometheus-textformat-merge/actions/workflows/release.yaml/badge.svg)](https://github.com/hansmi/prometheus-textformat-merge/actions/workflows/release.yaml)
[![CI workflow](https://github.com/hansmi/prometheus-textformat-merge/actions/workflows/ci.yaml/badge.svg)](https://github.com/hansmi/prometheus-textformat-merge/actions/workflows/ci.yaml)
[![Go reference](https://pkg.go.dev/badge/github.com/hansmi/prometheus-textformat-merge.svg)](https://pkg.go.dev/github.com/hansmi/prometheus-textformat-merge)

`prometheus-textformat-merge` is a command line program to combine multiple
[Prometheus `textformat`][prom_textformat] inputs.

Prometheus' [node exporter][node_exporter_doc] has a `textfile` collector
reading textformat files in a predetermined directory. When multiple files
contain the same metrics, albeit with different labels, collection fails
(see also [prometheus/node\_exporter#1885][node_exporter_issue1885]).

There are other use cases where combining multiple metrics sources is useful,
e.g. after downloading them from a collector using [cURL][curl].

The following inputs are supported:

* Regular files using the Prometheus text format
* Standard input
* Directories with multiple files with the `--dirs` flag (enumerates `*.prom`
  in the given directories by default)

## Example usage

```bash
$ cat >first.prom <<'EOF'
# HELP node_disk_io_time_seconds_total Total seconds spent doing I/Os.
# TYPE node_disk_io_time_seconds_total counter
node_disk_io_time_seconds_total{device="dm-0"} 581.412
node_disk_io_time_seconds_total{device="dm-1"} 483.348
EOF

$ cat >second.prom <<'EOF'
# HELP node_load5 5m load average.
# TYPE node_load5 gauge
node_load5 0.42
EOF

$ prometheus-textformat-merge first.prom second.prom
# HELP node_disk_io_time_seconds_total Total seconds spent doing I/Os.
# TYPE node_disk_io_time_seconds_total counter
node_disk_io_time_seconds_total{device="dm-0"} 581.412
node_disk_io_time_seconds_total{device="dm-1"} 483.348
# HELP node_load5 5m load average.
# TYPE node_load5 gauge
node_load5 0.42
```

Reading from standard input is also supported with the `-` placeholder:

```bash
$ prometheus-textformat-merge --output all.prom first.prom - <<'EOF'
# TYPE node_disk_io_time_seconds_total counter
node_disk_io_time_seconds_total{device="dm-4"} 104.156
node_disk_io_time_seconds_total{device="dm-5"} 0.372
EOF

$ cat all.prom
# HELP node_disk_io_time_seconds_total Total seconds spent doing I/Os.
# TYPE node_disk_io_time_seconds_total counter
node_disk_io_time_seconds_total{device="dm-0"} 581.412
node_disk_io_time_seconds_total{device="dm-1"} 483.348
node_disk_io_time_seconds_total{device="dm-4"} 104.156
node_disk_io_time_seconds_total{device="dm-5"} 0.372
```

Note how the same metric was combined from multiple sources and written to
a file. See the `--help` output for available flags.

## Installation

Pre-built binaries are provided for all [releases][releases]:

* Binary archives (`.tar.gz`)
* Debian/Ubuntu (`.deb`)
* RHEL/Fedora (`.rpm`)

With the source being available it's also possible to produce custom builds
directly using [Go][golang] or [GoReleaser][goreleaser].

[node_exporter_doc]: https://prometheus.io/docs/guides/node-exporter/
[node_exporter_issue1885]: https://github.com/prometheus/node_exporter/issues/1885
[prom_textformat]: https://prometheus.io/docs/instrumenting/exposition_formats/
[curl]: https://curl.se/
[releases]: https://github.com/hansmi/prometheus-textformat-merge/releases/latest
[golang]: https://golang.org/
[goreleaser]: https://goreleaser.com/

<!-- vim: set sw=2 sts=2 et : -->
