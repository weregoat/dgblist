package main

import (
	"fmt"
	"runtime"
	"time"
)

type Stats struct {
	Started   time.Time
	BytesRead int64
	IPAdded   int
	Events    int
	Interval  time.Duration
}

func (source *Source) LogStats() {
	now := time.Now()
	source.Info(
		fmt.Sprintf(
			"Running time: %s", now.Sub(source.Stats.Started),
		),
	)
	source.Info(
		fmt.Sprintf(
			"Bytes read: %d", source.Stats.BytesRead,
		),
	)
	source.Info(
		fmt.Sprintf("Addresses added to @%s: %d",
			source.Set.Name, source.Stats.IPAdded,
		),
	)
	source.Info(
		fmt.Sprintf("Events received: %d",
			source.Stats.Events,
		),
	)
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	source.Debug(
		fmt.Sprintf("Alloc = %v MiB", m.Alloc/(1024*1024)),
	)
	source.Debug(
		fmt.Sprintf("Total alloc = %v MiB", m.TotalAlloc/(1024*1024)),
	)
	source.Debug(
		fmt.Sprintf("Sys = %v MiB", m.Sys/(1024*1024)),
	)
	source.Debug(
		fmt.Sprintf("NumGC = %v MiB", m.NumGC),
	)
}
