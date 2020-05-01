package main

import (
	"fmt"
	"os"
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
			"%s running time: %s",
			source.Name,
			now.Sub(source.Stats.Started),
		),
	)
	source.Info(
		fmt.Sprintf(
			"%s bytes read: %d",
			source.Name,
			source.Stats.BytesRead,

		),
	)
	source.Info(
		fmt.Sprintf("%s addresses added to @%s: %d",
			source.Name,
			source.Set.Name,
			source.Stats.IPAdded,
		),
	)
	source.Info(
		fmt.Sprintf("%s events received: %d",
			source.Name,
			source.Stats.Events,
		),
	)
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	source.Debug(
		fmt.Sprintf("%s Alloc = %v MiB",
			os.Args[0],
			m.Alloc/(1024*1024)),
	)
	source.Debug(
		fmt.Sprintf( "%s Total alloc = %v MiB",
			os.Args[0],
			m.TotalAlloc/(1024*1024)),
	)
	source.Debug(
		fmt.Sprintf("%s Sys = %v MiB",
			os.Args[0],
			m.Sys/(1024*1024)),
	)
	source.Debug(
		fmt.Sprintf("%s NumGC = %v MiB",
			os.Args[0],
			m.NumGC),
	)
}
