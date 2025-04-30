package cmd

import (
	"slices"
	"time"
)

type metrics struct {
	Counters struct {
		Files struct {
			Fetched     int
			Partitioned int
			Skipped     int
		}
		Pages int
	}
	Durations struct {
		FetchKeys          time.Duration
		GetKeysToPartition time.Duration
		BuildSqsPayload    time.Duration
		SendSqsPayload     time.Duration
		Total              time.Duration
	}
	Timestamps struct {
		Start time.Time
	}
	Prefixes []string
}

func collectMetrics(ch chan metrics, timeStart time.Time) metrics {
	m := metrics{}
	m.Timestamps.Start = timeStart

	for {
		metric, ok := <-ch

		if !ok {
			break
		}

		m.Counters.Files.Fetched += metric.Counters.Files.Fetched
		m.Counters.Files.Partitioned += metric.Counters.Files.Partitioned
		m.Counters.Files.Skipped += metric.Counters.Files.Skipped
		m.Counters.Pages += metric.Counters.Pages

		for _, prefix := range metric.Prefixes {
			if !slices.Contains(m.Prefixes, prefix) {
				m.Prefixes = append(m.Prefixes, prefix)
			}
		}
		printProgress(&m)
	}

	return m
}
