package provider

import (
	"context"

	"github.com/pivotal-cf/brokerapi"
)

//go:generate counterfeiter -o fakes/fake_service_provider.go . ServiceProvider
type ServiceProvider interface {
	Provision(context.Context, ProvisionData) (dashboardURL, operationData string, isAsync bool, err error)
	Deprovision(context.Context, DeprovisionData) (operationData string, isAsync bool, err error)
	Bind(context.Context, BindData) (binding brokerapi.Binding, err error)
	Unbind(context.Context, UnbindData) (unbinding brokerapi.UnbindSpec, err error)
	Update(context.Context, UpdateData) (operationData string, isAsync bool, err error)
	LastOperation(context.Context, LastOperationData) (state brokerapi.LastOperationState, description string, err error)
}
