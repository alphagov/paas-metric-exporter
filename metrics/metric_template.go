package metrics

import (
	"bytes"
	"text/template"
)

func render(t string, data interface{}) (string, error) {
	if t == "" {
		t = "{{.Metric}}"
	}

	var metric bytes.Buffer
	tmpl, err := template.New("metric").Parse(t)

	if err != nil {
		return "", err
	}

	err = tmpl.Execute(&metric, data)
	if err != nil {
		return "", err
	}

	return metric.String(), nil
}
