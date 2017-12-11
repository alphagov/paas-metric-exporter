package processors

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/alphagov/paas-metric-exporter/events"
	"github.com/alphagov/paas-metric-exporter/metrics"
	sonde_events "github.com/cloudfoundry/sonde-go/events"
)

type LogMessageProcessor struct {
	tmpl string
}

func NewLogMessageProcessor(tmpl string) *LogMessageProcessor {
	return &LogMessageProcessor{tmpl: tmpl}
}

func (p *LogMessageProcessor) Process(appEvent *events.AppEvent) ([]metrics.Metric, error) {
	processedMetrics := []metrics.Metric{}

	logMessage := appEvent.Envelope.GetLogMessage()
	if logMessage.GetSourceType() != "API" || logMessage.GetMessageType() != sonde_events.LogMessage_OUT {
		return processedMetrics, nil
	}
	if !bytes.HasPrefix(logMessage.Message, []byte("App instance exited with guid ")) {
		return processedMetrics, nil
	}

	payloadStartMarker := []byte(" payload: {")
	payloadStartMarkerPosition := bytes.Index(logMessage.Message, payloadStartMarker)
	if payloadStartMarkerPosition < 0 {
		return processedMetrics, fmt.Errorf("unable to find start of payload in app instance exit log: %s", logMessage.Message)
	}
	payloadStartPosition := payloadStartMarkerPosition + len(payloadStartMarker) - 1

	payload := logMessage.Message[payloadStartPosition:]
	payloadAsJson := bytes.Replace(payload, []byte("=>"), []byte(":"), -1)

	var logMessagePayload struct {
		Index  int    `json:"index"`
		Reason string `json:"reason"`
	}
	err := json.Unmarshal(payloadAsJson, &logMessagePayload)
	if err != nil {
		return processedMetrics, fmt.Errorf("unable to parse payload in app instance exit log: %s", err)
	}

	if logMessagePayload.Reason != "CRASHED" {
		return processedMetrics, nil
	}

	vars := metrics.NewVars(appEvent)
	vars.Metric = "crash"
	vars.Instance = fmt.Sprintf("%d", logMessagePayload.Index)
	metricStat, err := vars.RenderTemplate(p.tmpl)
	if err == nil {
		metric := metrics.NewCounterMetric(metricStat, 1)
		processedMetrics = append(processedMetrics, *metric)
	}

	return processedMetrics, err
}
