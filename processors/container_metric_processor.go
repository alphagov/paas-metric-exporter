package processors

import (
	"errors"
	"strconv"

	"github.com/alphagov/paas-metric-exporter/events"
	"github.com/alphagov/paas-metric-exporter/metrics"
)

type ContainerMetricProcessor struct{}

func (p *ContainerMetricProcessor) Process(appEvent *events.AppEvent) ([]metrics.Metric, error) {
	processedMetrics := make([]metrics.Metric, 3)
	var err error

	for i, metricType := range []string{"cpu", "mem", "dsk"} {
		processedMetrics[i], err = p.ProcessContainerMetric(metricType, appEvent)
		if err != nil {
			return nil, err
		}
	}
	return processedMetrics, nil
}

func (p *ContainerMetricProcessor) ProcessContainerMetric(metricType string, appEvent *events.AppEvent) (metrics.GaugeMetric, error) {
	containerMetricEvent := appEvent.Envelope.GetContainerMetric()

	metric := metrics.GaugeMetric{
		Instance:     strconv.Itoa(int(containerMetricEvent.GetInstanceIndex())),
		App:          appEvent.App.Name,
		GUID:         appEvent.App.Guid,
		CellId:       appEvent.Envelope.GetIndex(),
		Job:          appEvent.Envelope.GetJob(),
		Organisation: appEvent.App.SpaceData.Entity.OrgData.Entity.Name,
		Space:        appEvent.App.SpaceData.Entity.Name,
	}

	switch metricType {
	case "cpu":
		metric.Metric = "cpu"
		metric.Value = int64(containerMetricEvent.GetCpuPercentage())
	case "mem":
		metric.Metric = "memoryBytes"
		metric.Value = int64(containerMetricEvent.GetMemoryBytes())
	case "dsk":
		metric.Metric = "diskBytes"
		metric.Value = int64(containerMetricEvent.GetDiskBytes())
	default:
		return metric, errors.New("Unsupported metric type.")
	}

	return metric, nil
}
