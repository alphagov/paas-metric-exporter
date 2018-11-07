package events

import (
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry/sonde-go/events"
)

type AppEvent struct {
	Envelope *events.Envelope
	App      cfclient.App
}

type ServiceEvent struct {
	Envelope *loggregator_v2.Envelope
	Service  cfclient.Service
}
