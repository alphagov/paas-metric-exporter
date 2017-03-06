package main

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("UPS", func() {
	var (
		ups     *UPS
		upsJSON string
	)

	BeforeEach(func() {
		ups = &UPS{}
	})

	Describe("UPSBind", func() {
		Context("parse provided json", func() {
			BeforeEach(func() {
				upsJSON = `{"user-provided":[{
       "credentials": {
         "api_endpoint": "https://google.com",
         "password": "P@55w0rd-",
         "skip_ssl_validation": true,
         "statsd_endpoint": "statsd.example.com",
         "statsd_prefix": "asda.",
         "username": "username"
       },
       "syslog_drain_url": "",
       "volume_mounts": [
       ],
       "label": "user-provided",
       "name": "nozzle-creds",
       "tags": [
       ]
    }]}`

				json.Unmarshal([]byte(upsJSON), &ups)
			})

			It("should expect a slice", func() {
				Expect(ups.UserProvided).To(Not(BeEmpty()))
			})

			It("should expect correct name", func() {
				name := "nozzle-creds"
				Expect(ups.UserProvided[0].Name).To(Equal(name))
			})

			It("should NOT generate fake user provided struct", func() {
				Expect(ups.First().Generated).To(BeFalse())
			})

			It("should obtain correct string credential", func() {
				failed := "I've failed!"
				Expect(ups.First().Credentials.GetStringValue("Password", failed)).To(Equal("P@55w0rd-"))
			})

			It("should obtain correct bool credential", func() {
				failed := false
				Expect(ups.First().Credentials.GetBoolValue("SkipSslValidation", failed)).To(BeTrue())
			})

			It("should fallback to string default", func() {
				fallback := "teapot"
				Expect(ups.First().Credentials.GetStringValue("NotExistingStringParameter", fallback)).To(Equal(fallback))
			})

			It("should fallback to bool default", func() {
				fallback := true
				Expect(ups.First().Credentials.GetBoolValue("NotExistingBoolParameter", fallback)).To(Equal(fallback))
			})
		})

		Context("no json provided", func() {
			It("should generate fake user provided struct", func() {
				Expect(ups.First().Generated).To(BeTrue())
			})

			It("should fallback to string default", func() {
				fallback := "teapot"
				Expect(ups.First().Credentials.GetStringValue("Username", fallback)).To(Equal(fallback))
			})

			It("should fallback to bool default", func() {
				fallback := true
				Expect(ups.First().Credentials.GetBoolValue("SkipSslValidation", fallback)).To(Equal(fallback))
			})
		})
	})
})
