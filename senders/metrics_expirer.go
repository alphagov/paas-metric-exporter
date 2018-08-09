package senders

import (
	"sync"
	"time"

	"github.com/mitchellh/hashstructure"
	"github.com/prometheus/client_golang/prometheus"
)

type metricID struct {
	Name   string
	Labels prometheus.Labels
}

type metricActivity struct {
	metricID  metricID
	timestamp time.Time
}

type MetricsExpirer struct {
	mtx            sync.Mutex
	metricActivity map[uint64]metricActivity
	callback       MetricsExpirerCallback
	ttl            time.Duration

	quit chan struct{}
}

type MetricsExpirerCallback func(name string, labels prometheus.Labels)

func NewMetricsExpirer(
	callback MetricsExpirerCallback,
	ttl time.Duration,
	frequency time.Duration,
) *MetricsExpirer {

	e := &MetricsExpirer{
		metricActivity: map[uint64]metricActivity{},
		callback:       callback,
		ttl:            ttl,
		quit:           make(chan struct{}),
	}

	ticker := time.NewTicker(frequency)
	go func() {
		for {
			select {
			case <-ticker.C:
				e.expireMetrics()
			case <-e.quit:
				ticker.Stop()
				return
			}
		}
	}()

	return e
}

func (s *MetricsExpirer) Stop() {
	s.quit <- struct{}{}
}

func (s *MetricsExpirer) SeenMetric(name string, labels prometheus.Labels) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	id := metricID{
		Name:   name,
		Labels: labels,
	}
	h, _ := hashstructure.Hash(id, nil)
	s.metricActivity[h] = metricActivity{
		metricID:  id,
		timestamp: time.Now(),
	}
}

func (s *MetricsExpirer) expireMetrics() {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	cutTime := time.Now().Add(-1 * s.ttl)
	for h, a := range s.metricActivity {
		if cutTime.After(a.timestamp) {
			id := a.metricID
			s.callback(id.Name, id.Labels)
			delete(s.metricActivity, h)
		}
	}
}
