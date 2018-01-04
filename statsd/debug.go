package statsd

import (
	"log"
	"time"
)

type DebugClient struct {
	Prefix string
}

func (d DebugClient) Gauge(stat string, value int64) error {
	log.Printf("gauge %s %d\n", d.Prefix+stat, value)
	return nil
}

func (d DebugClient) FGauge(stat string, value float64) error {
	log.Printf("gauge %s %f\n", d.Prefix+stat, value)
	return nil
}

func (d DebugClient) Incr(stat string, count int64) error {
	log.Printf("incr %s %d\n", d.Prefix+stat, count)
	return nil
}

func (d DebugClient) Timing(stat string, delta int64) error {
	log.Printf("timing %s %d\n", d.Prefix+stat, delta)
	return nil
}

func (d DebugClient) PrecisionTiming(stat string, delta time.Duration) error {
	log.Printf("timing %s %d\n", d.Prefix+stat, delta)
	return nil
}
