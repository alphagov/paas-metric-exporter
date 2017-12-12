package processors

import (
	"fmt"
	"time"

	"github.com/alphagov/paas-metric-exporter/events"
	"github.com/alphagov/paas-metric-exporter/metrics"
	sonde_events "github.com/cloudfoundry/sonde-go/events"
)

type HttpStartStopProcessor struct {
	tmpl string
}

func NewHttpStartStopProcessor(tmpl string) *HttpStartStopProcessor {
	return &HttpStartStopProcessor{tmpl: tmpl}
}

func (p *HttpStartStopProcessor) Process(appEvent *events.AppEvent) ([]metrics.Metric, error) {
	httpStartStop := appEvent.Envelope.GetHttpStartStop()
	if httpStartStop.GetPeerType() != sonde_events.PeerType_Client {
		return []metrics.Metric{}, nil
	}

	requestCountMetric, err := p.requestCount(httpStartStop, metrics.NewVars(appEvent))
	if err != nil {
		return []metrics.Metric{}, nil
	}
	responseTimeMetric, err := p.responseTime(httpStartStop, metrics.NewVars(appEvent))
	if err != nil {
		return []metrics.Metric{}, nil
	}
	return []metrics.Metric{requestCountMetric, responseTimeMetric}, nil
}

func (p *HttpStartStopProcessor) requestCount(httpStartStop *sonde_events.HttpStartStop, vars *metrics.Vars) (metrics.Metric, error) {
	vars.Metric = "requests." + statusClass(int(*httpStartStop.StatusCode))
	vars.Instance = fmt.Sprintf("%d", *httpStartStop.InstanceIndex)
	metricStat, err := vars.RenderTemplate(p.tmpl)
	if err != nil {
		return nil, err
	}
	return metrics.NewCounterMetric(metricStat, 1), nil
}

func (p *HttpStartStopProcessor) responseTime(httpStartStop *sonde_events.HttpStartStop, vars *metrics.Vars) (metrics.Metric, error) {
	vars.Metric = "responseTime." + statusClass(int(*httpStartStop.StatusCode))
	vars.Instance = fmt.Sprintf("%d", *httpStartStop.InstanceIndex)
	metricStat, err := vars.RenderTemplate(p.tmpl)
	if err != nil {
		return nil, err
	}

	startTimestamp := httpStartStop.GetStartTimestamp()
	stopTimestamp := httpStartStop.GetStopTimestamp()
	elapsedDuration := time.Duration(stopTimestamp - startTimestamp)
	return metrics.NewPrecisionTimingMetric(metricStat, elapsedDuration), nil
}

func statusClass(statusCode int) string {
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
