package metrics

import (
	"bytes"
	"text/template"

	"github.com/alphagov/paas-cf-apps-statsd/events"
)

// Vars will contain the variables the tenant could use to compose their
// custom metric namespace.
type Vars struct {
	App          string
	CellId       string
	GUID         string
	Index        string
	Instance     string
	Job          string
	Metric       string // cpu, memory, disk
	Organisation string
	Space        string
}

// Compose the new metric from all given data.
func (mv *Vars) Compose(providedTemplate string) (string, error) {
	var metric bytes.Buffer
	tmpl, err := template.New("metric").Parse(providedTemplate)

	if err != nil {
		return "", err
	}

	err = tmpl.Execute(&metric, mv)
	if err != nil {
		return "", err
	}

	return metric.String(), nil
}

func (mv *Vars) Parse(stream *events.AppEvent) {
	mv.App = stream.App.Name
	mv.GUID = stream.App.Guid
	mv.CellId = stream.Envelope.GetIndex()
	mv.Instance = ""
	mv.Job = stream.Envelope.GetJob()
	mv.Metric = ""
	mv.Organisation = stream.App.SpaceData.Entity.OrgData.Entity.Name
	mv.Space = stream.App.SpaceData.Entity.Name
}
