package config_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"

	"code.cloudfoundry.org/debugserver"
	loggingclient "code.cloudfoundry.org/diego-logging-client"
	"code.cloudfoundry.org/durationjson"
	"code.cloudfoundry.org/lager/lagerflags"
	"code.cloudfoundry.org/locket/cmd/locket/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LocketConfig", func() {
	var configFilePath, configData string

	BeforeEach(func() {
		configData = `{
			"log_level": "debug",
			"listen_address": "1.2.3.4:9090",
			"database_driver": "mysql",
			"max_open_database_connections": 1000,
			"database_connection_string": "stuff",
			"debug_address": "some-more-stuff",
			"consul_cluster": "http://127.0.0.1:1234,http://127.0.0.1:12345",
			"ca_file": "i am a ca file",
			"cert_file": "i am a cert file",
			"enable_consul_service_registration": false,
			"key_file": "i am a key file",
			"sql_ca_cert_file": "/var/vcap/jobs/locket/config/sql.ca",
      "report_interval":"1s",
			"loggregator": {
				"loggregator_use_v2_api": true,
				"loggregator_api_port": 1234,
				"loggregator_ca_path": "/var/ca_cert",
				"loggregator_cert_path": "/var/cert_path",
				"loggregator_key_path": "/var/key_path",
				"loggregator_source_id": "my-source-id",
				"loggregator_instance_id": "1",
				"loggregator_job_origin": "Earth"
			}
		}`
	})

	JustBeforeEach(func() {
		configFile, err := ioutil.TempFile("", "config-file")
		Expect(err).NotTo(HaveOccurred())

		n, err := configFile.WriteString(configData)
		Expect(err).NotTo(HaveOccurred())
		Expect(n).To(Equal(len(configData)))

		configFilePath = configFile.Name()
	})

	AfterEach(func() {
		err := os.RemoveAll(configFilePath)
		Expect(err).NotTo(HaveOccurred())
	})

	It("correctly parses the config file", func() {
		locketConfig, err := config.NewLocketConfig(configFilePath)
		Expect(err).NotTo(HaveOccurred())

		config := config.LocketConfig{
			DatabaseDriver:                  "mysql",
			ListenAddress:                   "1.2.3.4:9090",
			DatabaseConnectionString:        "stuff",
			MaxOpenDatabaseConnections:      1000,
			ConsulCluster:                   "http://127.0.0.1:1234,http://127.0.0.1:12345",
			EnableConsulServiceRegistration: false,
			LagerConfig: lagerflags.LagerConfig{
				LogLevel: "debug",
			},
			DebugServerConfig: debugserver.DebugServerConfig{
				DebugAddress: "some-more-stuff",
			},
			CaFile:        "i am a ca file",
			CertFile:      "i am a cert file",
			KeyFile:       "i am a key file",
			SQLCACertFile: "/var/vcap/jobs/locket/config/sql.ca",
			LoggregatorConfig: loggingclient.Config{
				UseV2API:   true,
				APIPort:    1234,
				CACertPath: "/var/ca_cert",
				CertPath:   "/var/cert_path",
				KeyPath:    "/var/key_path",
				JobOrigin:  "Earth",
				SourceID:   "my-source-id",
				InstanceID: "1",
			},
			ReportInterval: durationjson.Duration(time.Second),
		}

		Expect(locketConfig).To(Equal(config))
	})

	Context("when the file does not exist", func() {
		It("returns an error", func() {
			_, err := config.NewLocketConfig("foobar")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when the file does not contain valid json", func() {
		BeforeEach(func() {
			configData = "{{"
		})

		It("returns an error", func() {
			_, err := config.NewLocketConfig(configFilePath)
			Expect(err).To(HaveOccurred())
		})

	})

	Context("default values", func() {
		BeforeEach(func() {
			configData = `{}`
		})

		It("uses default values when they are not specified", func() {
			locketConfig, err := config.NewLocketConfig(configFilePath)
			Expect(err).NotTo(HaveOccurred())

			Expect(locketConfig).To(Equal(config.DefaultLocketConfig()))
		})

		Context("when serialized from LocketConfig", func() {
			Context("when no values are specified", func() {
				BeforeEach(func() {
					locketConfig := config.LocketConfig{}
					bytes, err := json.Marshal(locketConfig)
					Expect(err).NotTo(HaveOccurred())
					configData = string(bytes)
				})

				It("uses default values when they are not specified", func() {
					locketConfig, err := config.NewLocketConfig(configFilePath)
					Expect(err).NotTo(HaveOccurred())

					Expect(locketConfig).To(Equal(config.DefaultLocketConfig()))
				})
			})

			Context("when some values are specified", func() {
				BeforeEach(func() {
					locketConfig := config.LocketConfig{
						EnableConsulServiceRegistration: false,
					}
					bytes, err := json.Marshal(locketConfig)
					Expect(err).NotTo(HaveOccurred())
					configData = string(bytes)
				})

				It("does not overwrite specified values with default values", func() {
					locketConfig, err := config.NewLocketConfig(configFilePath)
					Expect(err).NotTo(HaveOccurred())
					Expect(locketConfig.EnableConsulServiceRegistration).To(Equal(false))
				})
			})
		})
	})
})
