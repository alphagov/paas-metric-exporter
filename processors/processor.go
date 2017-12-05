package processors

import (
	"github.com/alphagov/paas-cf-apps-statsd/events"
	"github.com/alphagov/paas-cf-apps-statsd/metrics"
)

type Processor interface {
	Process(event *events.AppEvent) ([]metrics.Metric, error)
}
