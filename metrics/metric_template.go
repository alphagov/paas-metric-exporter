package metrics

import (
	"bytes"
	"text/template"
)

// func NewVars(appEvent *events.AppEvent) *Vars {
// 	return &Vars{
// 		App:          appEvent.App.Name,
// 		GUID:         appEvent.App.Guid,
// 		CellId:       appEvent.Envelope.GetIndex(),
// 		Instance:     "",
// 		Job:          appEvent.Envelope.GetJob(),
// 		Metric:       "",
// 		Organisation: appEvent.App.SpaceData.Entity.OrgData.Entity.Name,
// 		Space:        appEvent.App.SpaceData.Entity.Name,
// 	}
// }

func render(t string, data interface{}) string {
	if t == "" {
		t = "{{.Metric}}"
	}

	var metric bytes.Buffer
	tmpl, err := template.New("metric").Parse(t)

	if err != nil {
		panic(err) // FIXME
	}

	err = tmpl.Execute(&metric, data)
	if err != nil {
		panic(err) // FIXME
	}

	return metric.String()
}
