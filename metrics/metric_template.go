package metrics

import (
	"bytes"
	"text/template"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/sonde-go/events"
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

type Stream struct {
	Msg  *events.Envelope
	App  cfclient.App
	Tmpl string
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

func (mv *Vars) Parse(stream *Stream) {
	space, _ := stream.App.Space()
	org, _ := space.Org()

	mv.App = stream.App.Name
	mv.GUID = stream.App.Guid
	mv.CellId = stream.Msg.GetIndex()
	mv.Instance = ""
	mv.Job = stream.Msg.GetJob()
	mv.Metric = ""
	mv.Organisation = org.Name
	mv.Space = space.Name
}
