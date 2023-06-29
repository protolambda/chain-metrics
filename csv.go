package main

import (
	"context"
	"fmt"
	"io"
	"strconv"
)

func WriteCSVMetrics[E any](ctx context.Context, timeFn func(elem E) int64, aggMetric AggregateMetric[E], w io.Writer, elems <-chan E) error {
	dest := make([]float64, len(aggMetric.Names), len(aggMetric.Names))
	buf := make([]byte, (1+len(dest))*10)
	line := 0
	for {
		// clean the array
		for i := range dest {
			dest[i] = 0
		}
		// reset the write buffer
		buf = buf[:0]
		// get the next element
		select {
		case <-ctx.Done():
			return ctx.Err()
		case elem, ok := <-elems:
			if !ok {
				return nil
			}
			t := timeFn(elem)
			// collect metrics values
			if err := aggMetric.Fn(elem, dest); err != nil {
				return fmt.Errorf("failed to collect line %d (t=%d) metric: %w", line, t, err)
			}
			// write timestamp
			buf = strconv.AppendInt(buf, t, 10)
			// format metrics values and write to buffer
			for _, v := range dest {
				buf = append(buf, ',')
				buf = strconv.AppendFloat(buf, v, 'f', 8, 64)
			}
			buf = append(buf, '\n')
			// write to output
			if _, err := w.Write(buf); err != nil {
				return fmt.Errorf("failed to write line %d (t=%d): %w", line, t, err)
			}
		}
		line += 1
	}
}
