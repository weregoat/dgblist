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
	LinesRead int64
	IPAdded   int
	Events    int
	Interval  time.Duration
}

func (source *Source) LogStats() {
	now := time.Now()
	source.Info(
		fmt.Sprintf("source %+q current log file: %s",
			source.Name,
			source.File.Name(),
		),
	)
	source.Info(
		fmt.Sprintf(
			"source %+q running time: %s",
			source.Name,
			now.Sub(source.Stats.Started),
		),
	)
	source.Info(
		fmt.Sprintf(
			"source %+q total read since start: %s",
			source.Name,
			formatBytes(source.Stats.BytesRead),
		),
	)
	source.Info(
		fmt.Sprintf(
			"source %+q bytes read current log file: %s",
			source.Name,
			formatBytes(source.Pos),
		),
	)
	source.Info(
		fmt.Sprintf(
			"source %+q lines processed: %d",
			source.Name,
			source.Stats.LinesRead,
		),
	)
	source.Info(
		fmt.Sprintf("source %+q addresses added to @%s: %d",
			source.Name,
			source.Set.Name,
			source.Stats.IPAdded,
		),
	)
	source.Info(
		fmt.Sprintf("source %+q events received: %d",
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
		fmt.Sprintf("%s Total alloc = %v MiB",
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

func formatBytes(bytes int64) string {
	var div int64 = 1
	var symbol = "B"
	symbols := map[int64]string{
		1000000000: "GB",
		1000000:    "MB",
		1000:       "kB",
	}
	for d, s := range symbols {
		if bytes > d {
			div = d
			symbol = s
		}
	}
	return fmt.Sprintf("%d %s", bytes/div, symbol)
}
