package events

import (
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/sonde-go/events"
)

type AppEvent struct {
	Envelope *events.Envelope
	App      cfclient.App
}
