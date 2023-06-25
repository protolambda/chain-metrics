package main

import (
	"fmt"
	"sort"
)

type Label struct {
	Key, Value string
}

type Metric[E any] struct {
	Name   string
	Labels []Label
	Fn     func(elem E) (float64, error)
}

func formatLabeledMetric(name string, labels []Label) string {
	out := name
	if len(labels) > 0 {
		out += "["
		for i, lab := range labels {
			out += lab.Key + "=" + lab.Value
			if i < len(labels)-1 {
				out += ","
			}
		}
		out += "]"
	}
	return out
}

func (m *Metric[E]) String() string {
	return formatLabeledMetric(m.Name, m.Labels)
}

type AggregateMetric[E any] struct {
	Names  []string
	Labels [][]Label
	Fn     func(elem E, dest []float64) error
}

func (m *AggregateMetric[E]) String() string {
	out := fmt.Sprintf("aggregate (%d):\n", len(m.Names))
	for i, name := range m.Names {
		out += "  " + formatLabeledMetric(name, m.Labels[i]) + "\n"
	}
	return out
}

func Histogram[E any](name string, bounds []float64, fn func(elem E, add func(v float64)) error) AggregateMetric[E] {
	sort.Float64s(bounds)
	// add 1 for the Infinity max bound
	n := len(bounds) + 1
	names := make([]string, n, n)
	for i := 0; i < n; i++ {
		names[i] = fmt.Sprintf("_bucket")
	}
	names = append(names, "_sum", "_count")
	sumIndex := n
	countIndex := n + 1

	labels := make([][]Label, 0, n+2)
	for _, b := range bounds {
		labels = append(labels, []Label{
			Label{Key: "le", Value: fmt.Sprintf("%f", b)},
		})
	}
	labels = append(labels, []Label{Label{Key: "le", Value: "Inf"}})
	// no labels on sum and count
	labels = append(labels, nil, nil)
	outFn := func(elem E, dest []float64) error {
		add := func(v float64) {
			i := sort.SearchFloat64s(bounds, v)
			dest[i] += 1
			dest[sumIndex] += v
			dest[countIndex] += 1
		}
		return fn(elem, add)
	}
	return AggregateMetric[E]{
		Names:  names,
		Labels: labels,
		Fn:     outFn,
	}
}

func Aggregate[E any](metrics ...Metric[E]) AggregateMetric[E] {
	names := make([]string, 0, len(metrics))
	labels := make([][]Label, 0, len(metrics))
	fn := func(elem E, dest []float64) error {
		var err error
		for i, m := range metrics {
			dest[i], err = m.Fn(elem)
			if err != nil {
				return fmt.Errorf("metric %s failed: %w", m.String(), err)
			}
		}
		return nil
	}
	return AggregateMetric[E]{
		Names:  names,
		Labels: labels,
		Fn:     fn,
	}
}

func CombineAggregates[E any](aggs ...AggregateMetric[E]) AggregateMetric[E] {
	n := 0
	for _, agg := range aggs {
		n += len(agg.Names)
	}
	names := make([]string, 0, n)
	labels := make([][]Label, 0, n)
	for _, agg := range aggs {
		names = append(names, agg.Names...)
		labels = append(labels, agg.Labels...)
	}
	fn := func(elem E, dest []float64) error {
		offset := 0
		for i, agg := range aggs {
			if err := agg.Fn(elem, dest[offset:offset+len(agg.Names)]); err != nil {
				return fmt.Errorf("agg %d failed: %w", i, err)
			}
			offset += len(agg.Names)
		}
		return nil
	}
	return AggregateMetric[E]{
		Names:  names,
		Labels: labels,
		Fn:     fn,
	}
}

func ParametrizedMetric[E any](name string, key string, values []string, fn func(elem E, dest []float64) error) AggregateMetric[E] {
	names := make([]string, len(values), len(values))
	for i := range names {
		names[i] = name
	}
	labels := make([][]Label, len(values), len(values))
	for i, v := range values {
		labels[i] = []Label{
			Label{Key: key, Value: v},
		}
	}
	outFn := func(elem E, dest []float64) error {
		return fn(elem, dest)
	}
	return AggregateMetric[E]{
		Names:  names,
		Labels: labels,
		Fn:     outFn,
	}
}

func TransformAggregate[A, B any](conv func(B) A, agg AggregateMetric[A]) AggregateMetric[B] {
	return AggregateMetric[B]{
		Names:  agg.Names,
		Labels: agg.Labels,
		Fn: func(elem B, dest []float64) error {
			a := conv(elem)
			return agg.Fn(a, dest)
		},
	}
}
