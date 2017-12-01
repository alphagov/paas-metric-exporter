# paas-cf-apps-statsd

This application consumes container metrics off the Cloud Foundry Doppler daemon, processes them based on the provided metrics template, and then sends them to a StatsD endpoint.

CPU, RAM and disk usage metrics for app containers will be sent through to StatsD as a gauge metric. It will get metrics for all apps that the user has access to.

The application is based on [`pivotal-cf/graphite-nozzle`](https://github.com/pivotal-cf/graphite-nozzle).

## Getting Started

Refer to the [PaaS Technical Documentation](https://docs.cloud.service.gov.uk/#exporting-metrics-data) for instructions on how to set up the metrics exporter app. Information on the configuration options are in the following table.


|Configuration Option|Application Flag|Environment Variable|Notes|
|:---|:---|:---|:---|
|API endpoint|api-endpoint|API_ENDPOINT||
|Statsd endpoint|statsd-endpoint|STATSD_ENDPOINT||
|Statsd prefix|statsd-prefix|STATSD_PREFIX||
|Username|username|USERNAME||
|Password|password|PASSWORD||
|Skip SSL Validation|skip-ssl-validation|SKIP_SSL_VALIDATION||
|Enable debug mode|debug|DEBUG|This disables forwarding to statsd and prints to stdout|
|Update frequency|update-frequency|UPDATE_FREQUENCY|The time in seconds, that takes between each apps update call|
|Metric template|metric-template|METRIC_TEMPLATE|The template that will form a new metric namespace|

## Supported template fields

You can use following template fields in your metric template:

* `{{.App}}` - name of the application
* `{{.CellId}}` - Cell GUID
* `{{.GUID}}` - Application ID
* `{{.Index}}` - BOSH job index e.g. `0`
* `{{.Instance}}` - Application instance
* `{{.Job}}` - BOSH job name e.g `cell`
* `{{.Metric}}` - cpu, memoryBytes or diskBytes
* `{{.Organisation}}` - a CF organisation that the app belongs to
* `{{.Space}}` - CF space used to deploy application 

## Testing

To run the test suite, first make sure you have ginkgo and gomega installed:

```
go get github.com/onsi/ginkgo/ginkgo
go get github.com/onsi/gomega
```

Then run `ginkgo -r` from root of this repository.
