package processors

import (
	"github.com/alphagov/paas-cf-apps-statsd/metrics"
)

type Processor interface {
	Process(stream *metrics.Stream) ([]metrics.Metric, error)
}
