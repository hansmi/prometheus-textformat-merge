package main

import (
	"context"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"

	dto "github.com/prometheus/client_model/go"
)

func TestMergeFamily(t *testing.T) {
	for _, tc := range []struct {
		name    string
		dst     *dto.MetricFamily
		src     *dto.MetricFamily
		want    *dto.MetricFamily
		wantErr *regexp.Regexp
	}{
		{
			name: "empty",
			dst:  &dto.MetricFamily{},
			src:  &dto.MetricFamily{},
			want: &dto.MetricFamily{},
		},
		{
			name: "empty dst",
			dst:  nil,
			src:  &dto.MetricFamily{Name: newString("src")},
			want: &dto.MetricFamily{Name: newString("src")},
		},
		{
			name:    "name mismatch",
			dst:     &dto.MetricFamily{Name: newString("foo")},
			src:     &dto.MetricFamily{Name: newString("bar")},
			wantErr: regexp.MustCompile(`^name mismatch\b`),
		},
		{
			name:    "type mismatch",
			dst:     &dto.MetricFamily{Name: newString("size"), Type: dto.MetricType_GAUGE.Enum()},
			src:     &dto.MetricFamily{Name: newString("size"), Type: dto.MetricType_COUNTER.Enum()},
			wantErr: regexp.MustCompile(`^type mismatch\b`),
		},
		{
			name: "help swap aa,bb",
			dst:  &dto.MetricFamily{Name: newString("count"), Help: newString("aa")},
			src:  &dto.MetricFamily{Name: newString("count"), Help: newString("bb")},
			want: &dto.MetricFamily{Name: newString("count"), Help: newString("aa")},
		},
		{
			name: "help swap bb,aa",
			dst:  &dto.MetricFamily{Name: newString("count"), Help: newString("bb")},
			src:  &dto.MetricFamily{Name: newString("count"), Help: newString("aa")},
			want: &dto.MetricFamily{Name: newString("count"), Help: newString("aa")},
		},
		{
			name: "empty src help",
			dst:  &dto.MetricFamily{Name: newString("count"), Help: newString("bb")},
			src:  &dto.MetricFamily{Name: newString("count")},
			want: &dto.MetricFamily{Name: newString("count"), Help: newString("bb")},
		},
		{
			name: "append metrics",
			dst: &dto.MetricFamily{Name: newString("xyz"), Metric: []*dto.Metric{
				{Gauge: &dto.Gauge{Value: newFloat64(1)}},
			}},
			src: &dto.MetricFamily{Name: newString("xyz"), Metric: []*dto.Metric{
				{Gauge: &dto.Gauge{Value: newFloat64(9)}},
				{Gauge: &dto.Gauge{Value: newFloat64(8)}},
			}},
			want: &dto.MetricFamily{Name: newString("xyz"), Metric: []*dto.Metric{
				{Gauge: &dto.Gauge{Value: newFloat64(1)}},
				{Gauge: &dto.Gauge{Value: newFloat64(9)}},
				{Gauge: &dto.Gauge{Value: newFloat64(8)}},
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := mergeFamily(tc.dst, tc.src)

			if tc.wantErr != nil {
				if err == nil || !tc.wantErr.MatchString(err.Error()) {
					t.Errorf("mergeFamily() failed with %v, want match for %q", err, tc.wantErr.String())
				}
			} else if err != nil {
				t.Errorf("mergeFamily() failed with %v", err)
			}

			if err == nil {
				if diff := cmp.Diff(got, tc.want, protocmp.Transform(), cmpopts.EquateEmpty()); diff != "" {
					t.Errorf("mergeFamily() difference (-got +want):\n%s", diff)
				}
			}
		})
	}
}

func TestMergedInputsWrite(t *testing.T) {
	for _, tc := range []struct {
		name   string
		names  bool
		inputs mergedInputs
		want   string
	}{
		{
			name: "empty",
		},
		{
			name:  "empty with names",
			names: true,
			want:  "# Sources:\n\n",
		},
		{
			name:  "two with names",
			names: true,
			inputs: mergedInputs{
				names: []string{"aaa.txt", "zzz.txt", "bbb.txt"},
				families: []*dto.MetricFamily{
					{
						Name: newString("first"),
						Type: dto.MetricType_COUNTER.Enum(),
						Help: newString("this is a test"),
						Metric: []*dto.Metric{
							{
								Counter: &dto.Counter{Value: newFloat64(13621)},
							},
						},
					},
					{
						Name: newString("requests"),
						Type: dto.MetricType_COUNTER.Enum(),
						Metric: []*dto.Metric{
							{
								Label: []*dto.LabelPair{
									{Name: newString("kind"), Value: newString("post")},
								},
								Counter: &dto.Counter{Value: newFloat64(14955)},
							},
							{
								Label: []*dto.LabelPair{
									{Name: newString("kind"), Value: newString("get")},
								},
								Counter: &dto.Counter{Value: newFloat64(18193)},
							},
						},
					},
				},
			},
			want: `# Sources:
# aaa.txt
# zzz.txt
# bbb.txt

# HELP first this is a test
# TYPE first counter
first 13621
# TYPE requests counter
requests{kind="post"} 14955
requests{kind="get"} 18193
`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf strings.Builder

			err := tc.inputs.write(&buf, tc.names)

			if err != nil {
				t.Errorf("mergedInputs() failed: %v", err)
			} else if diff := cmp.Diff(buf.String(), tc.want); diff != "" {
				t.Errorf("mergedInputs() difference (-got +want):\n%s", diff)
			}
		})
	}
}
func TestReadAndMerge(t *testing.T) {
	for _, tc := range []struct {
		name    string
		inputs  []inputWrapper
		want    *mergedInputs
		wantErr *regexp.Regexp
	}{
		{
			name: "empty",
			want: &mergedInputs{},
		},
		{
			name: "single gauge",
			inputs: []inputWrapper{
				newReaderInputWrapper(newFakeReaderWithName("a.txt",
					"# TYPE size GAUGE\nsize{kind=\"a\"} 1\n")),
			},
			want: &mergedInputs{
				names: []string{"a.txt"},
				families: []*dto.MetricFamily{
					{
						Name: newString("size"),
						Type: dto.MetricType_GAUGE.Enum(),
						Metric: []*dto.Metric{
							{
								Label: []*dto.LabelPair{
									{Name: newString("kind"), Value: newString("a")},
								},
								Gauge: &dto.Gauge{Value: newFloat64(1)},
							},
						},
					},
				},
			},
		},
		{
			name: "multiple inputs",
			inputs: []inputWrapper{
				newReaderInputWrapper(newFakeReaderWithName("a.txt",
					"# TYPE requests COUNTER\nrequests{kind=\"a\"} 26191\n")),
				newReaderInputWrapper(newFakeReaderWithName("b.txt",
					"# TYPE requests COUNTER\nrequests{kind=\"b\"} 15123\n"+
						"# TYPE size GAUGE\nsize 11477\n"+
						"# TYPE aborted COUNTER\naborted 5694\n")),
			},
			want: &mergedInputs{
				names: []string{"a.txt", "b.txt"},
				families: []*dto.MetricFamily{
					{
						Name: newString("aborted"),
						Type: dto.MetricType_COUNTER.Enum(),
						Metric: []*dto.Metric{
							{
								Counter: &dto.Counter{Value: newFloat64(5694)},
							},
						},
					},
					{
						Name: newString("requests"),
						Type: dto.MetricType_COUNTER.Enum(),
						Metric: []*dto.Metric{
							{
								Label: []*dto.LabelPair{
									{Name: newString("kind"), Value: newString("a")},
								},
								Counter: &dto.Counter{Value: newFloat64(26191)},
							},
							{
								Label: []*dto.LabelPair{
									{Name: newString("kind"), Value: newString("b")},
								},
								Counter: &dto.Counter{Value: newFloat64(15123)},
							},
						},
					},
					{
						Name: newString("size"),
						Type: dto.MetricType_GAUGE.Enum(),
						Metric: []*dto.Metric{
							{
								Gauge: &dto.Gauge{Value: newFloat64(11477)},
							},
						},
					},
				},
			},
		},
		{
			name: "type mismatch",
			inputs: []inputWrapper{
				newReaderInputWrapper(newFakeReaderWithName("a.txt",
					"# TYPE wrong GAUGE\nwrong 1\n")),
				newReaderInputWrapper(newFakeReaderWithName("b.txt",
					"# TYPE wrong COUNTER\nwrong 2\n")),
			},
			wantErr: regexp.MustCompile(`^family "wrong" from "b.txt": type mismatch\b`),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			got, err := readAndMerge(ctx, tc.inputs)

			if tc.wantErr != nil {
				if err == nil || !tc.wantErr.MatchString(err.Error()) {
					t.Errorf("readAndMerge() failed with %v, want match for %q", err, tc.wantErr.String())
				}
			} else if err != nil {
				t.Errorf("readAndMerge() failed with %v", err)
			}

			if err == nil {
				if diff := cmp.Diff(got, tc.want, protocmp.Transform(), cmp.AllowUnexported(mergedInputs{}), cmpopts.EquateEmpty(), cmpopts.EquateApprox(0, 0.0001)); diff != "" {
					t.Errorf("readAndMerge() difference (-got +want):\n%s", diff)
				}
			}
		})
	}
}

func TestReadAndMergeMany(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	var inputs []inputWrapper
	var wantNames []string
	var count = 100 + runtime.GOMAXPROCS(0)

	for i := 0; i < count; i++ {
		name := fmt.Sprintf("input%d", i)
		wantNames = append(wantNames, name)
		inputs = append(inputs, newReaderInputWrapper(newFakeReaderWithName(name,
			fmt.Sprintf("# TYPE test%[1]d GAUGE\ntest%[1]d %[1]d\n", i))))
	}

	got, err := readAndMerge(ctx, inputs)

	if err != nil {
		t.Errorf("readAndMerge() failed: %v", err)
	}

	if diff := cmp.Diff(got.names, wantNames); diff != "" {
		t.Errorf("names difference (-got +want):\n%s", diff)
	}

	if len(got.families) != count {
		t.Errorf("Result contained %d families, not %d: %v", len(got.families), count, got)
	}
}
