# paas-metric-exporter

This application consumes container metrics off the Cloud Foundry Doppler daemon, processes them based on the provided metrics template, and then sends them to a StatsD endpoint.

The application will get metrics for all apps that the user has access to.

The application is based on [`pivotal-cf/graphite-nozzle`](https://github.com/pivotal-cf/graphite-nozzle).

## Available metrics

The following metrics will be exported for every application instance.

|Name|Type|Description|
|:---|:---|:---|
|`cpu`|gauge|CPU utilisation|
|`diskBytes`|gauge|Disk usage in bytes|
|`memoryBytes`|gauge|Memory usage in bytes|
|`crash`|counter|Increased by one if the application crashed for any reason|
|`requests.1xx`|counter|Number of requests processed with 1xx status|
|`requests.2xx`|counter|Number of requests processed with 2xx status|
|`requests.3xx`|counter|Number of requests processed with 3xx status|
|`requests.4xx`|counter|Number of requests processed with 4xx status|
|`requests.5xx`|counter|Number of requests processed with 5xx status|
|`requests.other`|counter|Number of requests processed with unknown status|
|`responseTime.1xx`|timer|Timing of processed requests with 1xx status|
|`responseTime.2xx`|timer|Timing of processed requests with 2xx status|
|`responseTime.3xx`|timer|Timing of processed requests with 3xx status|
|`responseTime.4xx`|timer|Timing of processed requests with 4xx status|
|`responseTime.5xx`|timer|Timing of processed requests with 5xx status|
|`responseTime.other`|timer|Timing of processed requests with unknown status|

## Getting Started

Refer to the [PaaS Technical Documentation](https://docs.cloud.service.gov.uk/#metrics) for instructions on how to set up the metrics exporter app. Information on the configuration options are in the following table.

|Configuration Option|Application Flag|Environment Variable|Notes|
|:---|:---|:---|:---|
|API endpoint|api-endpoint|API_ENDPOINT||
|Statsd endpoint|statsd-endpoint|STATSD_ENDPOINT||
|Statsd prefix|statsd-prefix|STATSD_PREFIX|Namespace prepended to all emitted metric names. The default is `mycf` which results in metrics names like `mycf.cpu`, `mycf.diskBytes` etc|
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
* `{{.Metric}}` - a metric from the list of available metrics
* `{{.Organisation}}` - a CF organisation that the app belongs to
* `{{.Space}}` - CF space used to deploy application

## Testing

To run the test suite, first make sure you have ginkgo and gomega installed:

```
go get github.com/onsi/ginkgo/ginkgo
go get github.com/onsi/gomega
```

Then run `make test` from the root of this repository.

### Regenerating mocks

We generate some test mocks using counterfeiter. The mocks need to be regenerated if the mocked interfaces are changed.

To install counterfeiter please run first:
```
go get github.com/maxbrunsfeld/counterfeiter
```

To generate the mocks:
```
make generate
```
