package provider

import (
	"github.com/pivotal-cf/brokerapi"
)

type ProvisionData struct {
	InstanceID string
	Details    brokerapi.ProvisionDetails
	Service    brokerapi.Service
	Plan       brokerapi.ServicePlan
}

type DeprovisionData struct {
	InstanceID string
	Details    brokerapi.DeprovisionDetails
	Service    brokerapi.Service
	Plan       brokerapi.ServicePlan
}

type BindData struct {
	InstanceID   string
	BindingID    string
	Details      brokerapi.BindDetails
	AsyncAllowed bool
}

type UnbindData struct {
	InstanceID   string
	BindingID    string
	Details      brokerapi.UnbindDetails
	AsyncAllowed bool
}

type UpdateData struct {
	InstanceID string
	Details    brokerapi.UpdateDetails
	Service    brokerapi.Service
	Plan       brokerapi.ServicePlan
}

type LastOperationData struct {
	InstanceID  string
	PollDetails brokerapi.PollDetails
}
