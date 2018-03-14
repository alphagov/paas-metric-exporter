package presenters

import (
	"bytes"
	text_template "text/template"
)

type PathPresenter struct {
	tmpl *text_template.Template
}

func NewPathPresenter(template string) (PathPresenter, error) {
	if template == "" {
		template = "{{.Metric}}"
	}

	tmpl, err := text_template.New("metric").Parse(template)

	return PathPresenter{tmpl}, err
}

func (p PathPresenter) Present(data interface{}) (string, error) {
	var metric bytes.Buffer
	err := p.tmpl.Execute(&metric, data)

	if err != nil {
		return "", err
	}

	return metric.String(), nil
}
