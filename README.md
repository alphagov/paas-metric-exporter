# paas-cf-apps-statsd

This application consumes container metrics off the Cloud Foundry Doppler daemon, processes them based on provided metrics template, and then sends them off to a StatsD endpoint.
CPU, RAM and disk usage metrics for app containers will be sent through to StatsD as a Gauge metric. It will get metrics for all applications provided user has access to. Note that it is still being developed and shouldn't be considered production-ready.

The application is somewhat based on [`pivotal-cf/graphite-nozzle`](https://github.com/pivotal-cf/graphite-nozzle).

## Getting Started

* Clone this repository
* Deploy the app to CloudFoundry. You can use application flags or envirnoment variables to configure it.
  * Flags:
    * `--api-endpoint` - CloudFoundry API endpoint
    * `--statsd-endpoint` - Statsd endpoint
    * `--username` - UAA username
    * `--password` - UAA password
    * `--skip-ssl-validation` - Do not validate SSL certificate
    * `--debug` - Enable debug mode. This disables forwarding to statsd and prints to stdout
    * `--update-frequency` - The time in seconds, that takes between each apps update call.
    * `--metric-template` - The template that will form a new metric namespace. This uses [Go lang templating engine](https://golang.org/pkg/text/template/)

  * Example manifest:
```
---
applications:
- name: metric-exporter
  memory: 100M
  instances: 1
  buildpack: go_buildpack
  health-check-type: none
  no-route: true
  env:
    API_ENDPOINT: https://api.10.244.0.34.xip.io
    STATSD_ENDPOINT: 10.244.11.2:8125
    STATSD_PREFIX: myproject
    USERNAME: cloud_foundry_user
    PASSWORD: cloud_foundry_password
    SKIP_SSL_VALIDATION: false
    DEBUG: false
    UPDATE_FREQUENCY: 300
    METRIC_TEMPLATE: {{.Space}}.{{.App}}.{{.Metric}}
```

Be sure to provide credentials for the user assigned to the correct space(s), with ability to list applications.
We'd recommend using the `SpaceAuditor` role, which would fulfill the requirements of this app, and also
secure the environment to a certain level.

## Supported template fields

You can use following template fields in your metric template:

* `{{.App}}` - name of the application
* `{{.CellId}}` - Cell GUID
* `{{.GUID}}` - Application ID
* `{{.Instance}}` - Application instance
* `{{.Job}}` - BOSH kob name e.g `cell`
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
