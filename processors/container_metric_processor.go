package processors

import (
	"fmt"
	"strconv"

	"github.com/alphagov/paas-metric-exporter/events"
	"github.com/alphagov/paas-metric-exporter/metrics"
)

type ContainerMetricProcessor struct{}

func (p *ContainerMetricProcessor) Process(appEvent *events.AppEvent) ([]metrics.Metric, error) {
	metric := appEvent.Envelope.GetContainerMetric()

	if metric.GetMemoryBytesQuota() == 0 {
		return nil, fmt.Errorf("Memory quota is 0 for %s", appEvent.App.Guid)
	}

	if metric.GetDiskBytesQuota() == 0 {
		return nil, fmt.Errorf("Disk byte quota is 0 for %s", appEvent.App.Guid)
	}

	memoryUtilization := int64(float64(metric.GetMemoryBytes()) / float64(metric.GetMemoryBytesQuota()) * 100)
	diskUtilization := int64(float64(metric.GetDiskBytes()) / float64(metric.GetDiskBytesQuota()) * 100)

	return []metrics.Metric{
		createContainerMetric(appEvent, "cpu", int64(metric.GetCpuPercentage()), "percentage"),
		createContainerMetric(appEvent, "memoryBytes", int64(metric.GetMemoryBytes()), "bytes"),
		createContainerMetric(appEvent, "memoryUtilization", memoryUtilization, "percentage"),
		createContainerMetric(appEvent, "diskBytes", int64(metric.GetDiskBytes()), "bytes"),
		createContainerMetric(appEvent, "diskUtilization", diskUtilization, "percentage"),
	}, nil
}

func createContainerMetric(appEvent *events.AppEvent, metric string, value int64, unit string) metrics.GaugeMetric {
	return metrics.GaugeMetric{
		Instance:     strconv.Itoa(int(appEvent.Envelope.GetContainerMetric().GetInstanceIndex())),
		App:          appEvent.App.Name,
		GUID:         appEvent.App.Guid,
		CellId:       appEvent.Envelope.GetIndex(),
		Job:          appEvent.Envelope.GetJob(),
		Organisation: appEvent.App.SpaceData.Entity.OrgData.Entity.Name,
		Space:        appEvent.App.SpaceData.Entity.Name,
		Metric:       metric,
		Value:        value,
		Unit:         unit,
	}
}
