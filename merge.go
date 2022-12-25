package main

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/prometheus/common/expfmt"
	"golang.org/x/sync/errgroup"

	dto "github.com/prometheus/client_model/go"
)

// mergeFamily combines two metric families. The destination is modified and returned.
func mergeFamily(dst *dto.MetricFamily, src *dto.MetricFamily) (*dto.MetricFamily, error) {
	if dst == nil {
		return src, nil
	}

	if dst.GetName() != src.GetName() {
		return nil, fmt.Errorf("name mismatch (dst is %q, src is %q)", dst.GetName(), src.GetName())
	}

	if dst.GetType() != src.GetType() {
		return nil, fmt.Errorf("type mismatch (dst is %v, src is %v)", dst.GetType(), src.GetType())
	}

	if dst.GetHelp() > src.GetHelp() && len(strings.TrimSpace(src.GetHelp())) > 0 {
		dst.Help = src.Help
	}

	dst.Metric = append(dst.Metric, src.Metric...)

	return dst, nil
}

type mergedInputs struct {
	names    []string
	families []*dto.MetricFamily
}

func (c *mergedInputs) write(w io.Writer, includeNames bool) error {
	if includeNames {
		io.WriteString(w, "# Sources:\n")

		for _, i := range c.names {
			fmt.Fprintf(w, "# %s\n", i)
		}

		io.WriteString(w, "\n")
	}

	for _, mf := range c.families {
		if _, err := expfmt.MetricFamilyToText(w, mf); err != nil {
			return fmt.Errorf("%s: %w", mf.GetName(), err)
		}
	}

	return nil
}

type metricsMerger struct {
	inputNames []string
	byName     map[string]*dto.MetricFamily
}

func newMetricsMerger() *metricsMerger {
	return &metricsMerger{
		byName: make(map[string]*dto.MetricFamily),
	}
}

func (m *metricsMerger) append(input parsedInput) error {
	m.inputNames = append(m.inputNames, input.name)

	for _, mf := range input.families {
		var err error

		name := mf.GetName()

		m.byName[name], err = mergeFamily(m.byName[name], mf)

		if err != nil {
			return fmt.Errorf("family %q from %q: %w", name, input.name, err)
		}
	}

	return nil
}

func (m *metricsMerger) finalize() *mergedInputs {
	families := make([]*dto.MetricFamily, 0, len(m.byName))

	for _, mf := range m.byName {
		families = append(families, mf)
	}

	// Sort by name
	sort.Slice(families, func(a, b int) bool {
		return families[a].GetName() < families[b].GetName()
	})

	return &mergedInputs{
		names:    m.inputNames,
		families: families,
	}
}

func mergeInputs(ctx context.Context, inputsCh <-chan parsedInput) (*mergedInputs, error) {
	merger := newMetricsMerger()

	for cur := range inputsCh {
		if err := merger.append(cur); err != nil {
			return nil, err
		}
	}

	return merger.finalize(), nil
}

func readAndMerge(ctx context.Context, inputs []inputWrapper) (*mergedInputs, error) {
	g, ctx := errgroup.WithContext(ctx)

	parsedCh := make(chan parsedInput)

	g.Go(func() error {
		defer close(parsedCh)

		return readInputs(ctx, inputs, parsedCh)
	})

	var merged *mergedInputs

	g.Go(func() error {
		var err error
		merged, err = mergeInputs(ctx, parsedCh)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return merged, nil
}
