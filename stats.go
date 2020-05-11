package main

import (
	"fmt"
	"runtime"
	"time"
)

type Stats struct {
	Started   time.Time
	BytesRead uint64
	LinesRead uint64
	IPAdded   int
	Events    int
	Interval  time.Duration
}

func (source *Source) LogStats() {
	now := time.Now()
	source.Debug(
		fmt.Sprintf("source %+q current log file: %s",
			source.Name,
			source.File.Name(),
		),
	)
	source.Debug(
		fmt.Sprintf(
			"source %+q running time: %s",
			source.Name,
			now.Sub(source.Stats.Started),
		),
	)
	source.Debug(
		fmt.Sprintf(
			"source %+q total read since start: %s",
			source.Name,
			formatBytes(source.Stats.BytesRead),
		),
	)
	source.Debug(
		fmt.Sprintf(
			"source %+q bytes read current log file: %s",
			source.Name,
			formatBytes(uint64(source.Pos)),
		),
	)
	source.Debug(
		fmt.Sprintf(
			"source %+q lines processed: %d",
			source.Name,
			source.Stats.LinesRead,
		),
	)
	source.Debug(
		fmt.Sprintf("source %+q addresses added to @%s: %d",
			source.Name,
			source.Set.Name,
			source.Stats.IPAdded,
		),
	)
	source.Debug(
		fmt.Sprintf("source %+q events received: %d",
			source.Name,
			source.Stats.Events,
		),
	)
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	source.Debug(
		fmt.Sprintf("statistics: heap alloc = %s",
			formatBytes(m.HeapAlloc),
		),
	)
	source.Debug(
		fmt.Sprintf("statistics: heap Idle = %s",
			formatBytes(m.HeapIdle),
		),
	)
	source.Debug(
		fmt.Sprintf("statistics: cumulative total alloc = %s",
			formatBytes(m.TotalAlloc),
		),
	)

	source.Debug(
		fmt.Sprintf("statistics: sys = %s",
			formatBytes(m.Sys)),
	)

	source.Debug(
		fmt.Sprintf("statistics: live objects = %d",
			m.Mallocs-m.Frees,
		),
	)

	source.Debug(
		fmt.Sprintf("statistics: GC compled cicles so far = %d",
			m.NumGC),
	)
	source.Debug(
		fmt.Sprintf("statistics: last GC at %s",
			time.Unix(0, int64(m.LastGC)).String(),
		),
	)
}

// formatBytes returns a string with the appropriate symbol for representing
// a quantity of bytes.
func formatBytes(bytes uint64) string {
	var div uint64 = 1
	var symbol = "B"
	symbols := map[uint64]string{
		1000000000: "GB",
		1000000:    "MB",
		1000:       "kB",
	}
	for d, s := range symbols {
		if bytes > d && d > div {
			div = d
			symbol = s
		}
	}
	return fmt.Sprintf("%d %s", bytes/div, symbol)
}
