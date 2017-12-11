package processors

import (
	"errors"
	"strconv"

	"github.com/alphagov/paas-metric-exporter/events"
	"github.com/alphagov/paas-metric-exporter/metrics"
)

type ContainerMetricProcessor struct {
	tmpl string
}

func NewContainerMetricProcessor(tmpl string) *ContainerMetricProcessor {
	return &ContainerMetricProcessor{tmpl: tmpl}
}

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
	instanceIndex := strconv.Itoa(int(containerMetricEvent.GetInstanceIndex()))

	var err error
	var metric metrics.GaugeMetric
	var metricStat string
	var value int64

	vars := metrics.NewVars(appEvent)
	vars.Instance = instanceIndex

	switch metricType {
	case "cpu":
		vars.Metric = "cpu"
		metricStat, err = vars.RenderTemplate(p.tmpl)
		value = int64(containerMetricEvent.GetCpuPercentage())
	case "mem":
		vars.Metric = "memoryBytes"
		metricStat, err = vars.RenderTemplate(p.tmpl)
		value = int64(containerMetricEvent.GetMemoryBytes())
	case "dsk":
		vars.Metric = "diskBytes"
		metricStat, err = vars.RenderTemplate(p.tmpl)
		value = int64(containerMetricEvent.GetDiskBytes())
	default:
		err = errors.New("Unsupported metric type.")
	}

	if err != nil {
		return metrics.GaugeMetric{}, err
	}

	metric = *metrics.NewGaugeMetric(metricStat, value)

	return metric, nil
}
