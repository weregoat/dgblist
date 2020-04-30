package main

import (
	"fmt"
	"time"
)

type Stats struct {
	Started   time.Time
	BytesRead int64
	IPAdded   int
	Events    int
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
		fmt.Sprintf("Address added to @%s: %d",
			source.Set.Name, source.Stats.IPAdded,
		),
	)
	source.Info(
		fmt.Sprintf("Events received: %d",
			source.Stats.Events,
		),
	)
}
