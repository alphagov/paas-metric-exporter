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

func NewVars(appEvent *events.AppEvent) *Vars {
	return &Vars{
		App:          appEvent.App.Name,
		GUID:         appEvent.App.Guid,
		CellId:       appEvent.Envelope.GetIndex(),
		Instance:     "",
		Job:          appEvent.Envelope.GetJob(),
		Metric:       "",
		Organisation: appEvent.App.SpaceData.Entity.OrgData.Entity.Name,
		Space:        appEvent.App.SpaceData.Entity.Name,
	}
}

// Compose the new metric from all given data.
func (v *Vars) RenderTemplate(providedTemplate string) (string, error) {
	var metric bytes.Buffer
	tmpl, err := template.New("metric").Parse(providedTemplate)

	if err != nil {
		return "", err
	}

	err = tmpl.Execute(&metric, v)
	if err != nil {
		return "", err
	}

	return metric.String(), nil
}
