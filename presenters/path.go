package presenters

import (
	"bytes"
	text_template "text/template"

	"github.com/alphagov/paas-metric-exporter/metrics"
)

type PathPresenter struct {
	tmpl *text_template.Template
}

type MetricView struct {
	App          string
	CellId       string
	GUID         string
	Instance     string
	Job          string
	Metric       string
	Organisation string
	Space        string
}

func NewPathPresenter(template string) (PathPresenter, error) {
	if template == "" {
		template = "{{.Metric}}"
	}

	tmpl, err := text_template.New("metric").Parse(template)

	return PathPresenter{tmpl}, err
}

func (p PathPresenter) Present(metric metrics.Metric) (string, error) {
	var buf bytes.Buffer
	labels := metric.GetLabels()

	metricName := metric.Name()

	// append statusRange to the metric name if present
	if statusRange, ok := labels["statusRange"]; ok {
		metricName += "." + statusRange
	}

	view := MetricView{
		App:          labels["App"],
		CellId:       labels["CellId"],
		GUID:         labels["GUID"],
		Instance:     labels["Instance"],
		Job:          labels["Job"],
		Metric:       metricName,
		Organisation: labels["Organisation"],
		Space:        labels["Space"],
	}
	err := p.tmpl.Execute(&buf, view)

	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
