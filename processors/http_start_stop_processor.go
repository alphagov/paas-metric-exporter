package processors

import (
	"fmt"
	"time"

	"github.com/alphagov/paas-metric-exporter/events"
	"github.com/alphagov/paas-metric-exporter/metrics"
	sonde_events "github.com/cloudfoundry/sonde-go/events"
)

type HttpStartStopProcessor struct{}

func (p *HttpStartStopProcessor) Process(appEvent *events.AppEvent) ([]metrics.Metric, error) {
	httpStartStop := appEvent.Envelope.GetHttpStartStop()
	if httpStartStop.GetPeerType() != sonde_events.PeerType_Client {
		return []metrics.Metric{}, nil
	}

	statusRange := statusRange(int(*httpStartStop.StatusCode))

	return []metrics.Metric{
		metrics.CounterMetric{
			Instance:     fmt.Sprintf("%d", *httpStartStop.InstanceIndex),
			App:          appEvent.App.Name,
			GUID:         appEvent.App.Guid,
			CellId:       appEvent.Envelope.GetIndex(),
			Job:          appEvent.Envelope.GetJob(),
			Organisation: appEvent.App.SpaceData.Entity.OrgData.Entity.Name,
			Space:        appEvent.App.SpaceData.Entity.Name,
			Metric:       "requests",
			Metadata:     map[string]string{"statusRange": statusRange},
			Value:        1,
		},
		metrics.PrecisionTimingMetric{
			Instance:     fmt.Sprintf("%d", *httpStartStop.InstanceIndex),
			App:          appEvent.App.Name,
			GUID:         appEvent.App.Guid,
			CellId:       appEvent.Envelope.GetIndex(),
			Job:          appEvent.Envelope.GetJob(),
			Organisation: appEvent.App.SpaceData.Entity.OrgData.Entity.Name,
			Space:        appEvent.App.SpaceData.Entity.Name,
			Metric:       "responseTime",
			Metadata:     map[string]string{"statusRange": statusRange},
			Value:        time.Duration(httpStartStop.GetStopTimestamp() - httpStartStop.GetStartTimestamp()),
		},
	}, nil
}

func statusRange(statusCode int) string {
	switch {
	case statusCode < 100:
		return "other"
	case statusCode < 200:
		return "1xx"
	case statusCode < 300:
		return "2xx"
	case statusCode < 400:
		return "3xx"
	case statusCode < 500:
		return "4xx"
	case statusCode < 600:
		return "5xx"
	default:
		return "other"
	}
}
