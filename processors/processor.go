package processors

import (
	"github.com/alphagov/paas-metric-exporter/events"
	"github.com/alphagov/paas-metric-exporter/metrics"
)

//go:generate counterfeiter -o mocks/processor.go . Processor
type Processor interface {
	Process(event *events.AppEvent) ([]metrics.Metric, error)
}

var _ Processor = &ContainerMetricProcessor{}

var _ Processor = &LogMessageProcessor{}

var _ Processor = &HttpStartStopProcessor{}
