package processors

import (
	"errors"
	"strconv"

	"github.com/alphagov/paas-cf-apps-statsd/metrics"
)

type ContainerMetricProcessor struct{}

func NewContainerMetricProcessor() *ContainerMetricProcessor {
	return &ContainerMetricProcessor{}
}

func (p *ContainerMetricProcessor) Process(stream *metrics.Stream) ([]metrics.Metric, error) {
	processedMetrics := make([]metrics.Metric, 3)
	var err error

	for i, metricType := range []string{"cpu", "mem", "dsk"} {
		processedMetrics[i], err = p.ProcessContainerMetric(metricType, stream)
		if err != nil {
			return nil, err
		}
	}
	return processedMetrics, nil
}

func (p *ContainerMetricProcessor) ProcessContainerMetric(metricType string, stream *metrics.Stream) (metrics.GaugeMetric, error) {
	containerMetricEvent := stream.Msg.GetContainerMetric()
	instanceIndex := strconv.Itoa(int(containerMetricEvent.GetInstanceIndex()))

	var err error
	var metric metrics.GaugeMetric
	var newMetric string
	var value int64

	mv := metrics.Vars{}
	mv.Parse(stream)

	mv.Instance = instanceIndex

	switch metricType {
	case "cpu":
		mv.Metric = "cpu"
		newMetric, err = mv.Compose(stream.Tmpl)
		value = int64(containerMetricEvent.GetCpuPercentage())
	case "mem":
		mv.Metric = "memoryBytes"
		newMetric, err = mv.Compose(stream.Tmpl)
		value = int64(containerMetricEvent.GetMemoryBytes())
	case "dsk":
		mv.Metric = "diskBytes"
		newMetric, err = mv.Compose(stream.Tmpl)
		value = int64(containerMetricEvent.GetDiskBytes())
	default:
		err = errors.New("Unsupported metric type.")
	}

	if err != nil {
		return metrics.GaugeMetric{}, err
	}

	metric = *metrics.NewGaugeMetric(newMetric, value)

	return metric, nil
}
