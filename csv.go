package main

import (
	"context"
	"fmt"
	"io"
	"strconv"
)

func WriteCSVMetrics[E any](ctx context.Context, aggMetric AggregateMetric[E], w io.Writer, elems <-chan E) error {
	dest := make([]float64, len(aggMetric.Names), len(aggMetric.Names))
	buf := make([]byte, len(dest)*10)
	line := 0
	for {
		// clean the array
		for i := range dest {
			dest[i] = 0
		}
		// clean the write buffer
		buf = buf[:0]
		// get the next element
		select {
		case <-ctx.Done():
			return ctx.Err()
		case elem, ok := <-elems:
			if !ok {
				return nil
			}
			// collect metrics values
			if err := aggMetric.Fn(elem, dest); err != nil {
				return fmt.Errorf("failed to collect line %d metric: %w", line, err)
			}
			// format metrics values and write to buffer
			for _, v := range dest {
				buf = strconv.AppendFloat(buf, v, 'f', 8, 64)
				buf = append(buf, ',')
			}
			// overwrite last comma
			buf[len(buf)-1] = '\n'
			// write to output
			if _, err := w.Write(buf); err != nil {
				return fmt.Errorf("failed to write line %d: %w", line, err)
			}
		}
		line += 1
	}
}
