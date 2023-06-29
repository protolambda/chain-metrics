package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
)

type MetricJSONEntry struct {
	Metric     json.RawMessage `json:"metric"`
	Values     []float64       `json:"values"`
	Timestamps json.RawMessage `json:"timestamps"`
}

func ExportJSONLines[E any](ctx context.Context, timeFn func(elem E) int64, aggMetric AggregateMetric[E], w io.Writer, elems <-chan E) error {

	n := 100
	// we can reuse the timestamps array between all metrics we write
	timestamps := make([]int64, 0, n)

	// temp buffer of values per element
	dest := make([]float64, len(aggMetric.Names), len(aggMetric.Names))

	var metrics []MetricJSONEntry
	for i, name := range aggMetric.Names {
		metric := make(map[string]string)
		metric["__name__"] = name
		for _, label := range aggMetric.Labels[i] {
			metric[label.Key] = label.Value
		}
		dat, err := json.Marshal(&metric)
		if err != nil {
			return fmt.Errorf("failed to encode metrics tags map of %q: %w", name, err)
		}
		metrics = append(metrics, MetricJSONEntry{
			Metric: json.RawMessage(dat),
			Values: make([]float64, 0, n),
		})
	}
	var timestampsBuf bytes.Buffer
	timestampsBuf.Grow(14 * n) // 13 bytes per timestamp, plus delimiters
	timestampsEnc := json.NewEncoder(&timestampsBuf)

	var outBuf bytes.Buffer
	jsonOut := json.NewEncoder(&outBuf)

	flush := func() error {
		if len(timestamps) == 0 {
			return nil // return early if there is nothing to output
		}
		// only encode timestamps once
		timestampsBuf.Reset()
		if err := timestampsEnc.Encode(timestamps); err != nil {
			return fmt.Errorf("failed to encode timestamps: %w", err)
		}
		timestampsData := json.RawMessage(timestampsBuf.Bytes())
		outBuf.Reset()
		for i := range metrics {
			metrics[i].Timestamps = timestampsData
			if err := jsonOut.Encode(&metrics[i]); err != nil {
				return fmt.Errorf("failed to encode metrics %d: %w", i, err)
			}
			outBuf.WriteByte('\n')
		}
		if _, err := w.Write(outBuf.Bytes()); err != nil {
			return fmt.Errorf("failed to write metrics (t0 = %d, count=%d) to output: %w", timestamps[0], len(timestamps), err)
		}
		// clear metrics
		for i := range metrics {
			metrics[i].Values = metrics[i].Values[:0]
		}
		// clear timestamps
		timestamps = timestamps[:0]
		return nil
	}

	for {
		// clean the array
		for i := range dest {
			dest[i] = 0
		}
		// get the next element
		select {
		case <-ctx.Done():
			return ctx.Err()
		case elem, ok := <-elems:
			if !ok {
				if err := flush(); err != nil {
					return fmt.Errorf("failed to flush metrics (on exit): %w", err)
				}
				return nil
			}
			t := timeFn(elem)
			// collect metrics values
			if err := aggMetric.Fn(elem, dest); err != nil {
				return fmt.Errorf("failed to collect t=%d metric: %w", t, err)
			}
			// append to destination metrics
			for i, v := range dest {
				metrics[i].Values = append(metrics[i].Values, v)
			}
			// append timestamp
			timestamps = append(timestamps, t)

			if len(timestamps) == n {
				if err := flush(); err != nil {
					return fmt.Errorf("failed to flush metrics: %w", err)
				}
			}
		}
	}
}
