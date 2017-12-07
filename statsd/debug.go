package statsd

import (
	"log"
	"time"
)

type DebugClient struct {
}

func (d DebugClient) Gauge(stat string, value int64) error {
	log.Printf("gauge %s %d\n", stat, value)
	return nil
}

func (d DebugClient) FGauge(stat string, value float64) error {
	log.Printf("gauge %s %f\n", stat, value)
	return nil
}

func (d DebugClient) Incr(stat string, count int64) error {
	log.Printf("incr %s %d\n", stat, count)
	return nil
}

func (d DebugClient) Timing(stat string, delta int64) error {
	log.Printf("timing %s %d\n", stat, delta)
	return nil
}

func (d DebugClient) PrecisionTiming(stat string, delta time.Duration) error {
	log.Printf("timing %s %d\n", stat, delta)
	return nil
}
