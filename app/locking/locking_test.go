package locking_test

import (
	"os/exec"
	"time"

	mockLocket "github.com/alphagov/paas-go/testing/fakes/locket/mock_server"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Locking", func() {
	var (
		mockLocketServer *mockLocket.MockLocketServer
		err              error
		fixturesPath     = "../../fixtures"
	)

	AfterEach(func() {
		mockLocketServer.Kill()
	})

	It("Should start cleanly if it can acquire a lock", func() {
		mockLocketServer = mockLocket.NewMockLocketServer(fixturesPath, "alwaysGrantLock")
		err = mockLocketServer.Run()
		Expect(err).NotTo(HaveOccurred())

		metricExporterSession, err := gexec.Start(
			exec.Command(
				metricExporterPath,
				"--debug",
				"--enable-locking",
				"--locket-address="+mockLocketServer.ListenAddress,
				"--locket-ca-cert="+fixturesPath+"/ca.cert.pem",
				"--locket-client-cert="+fixturesPath+"/client.cert.pem",
				"--locket-client-key="+fixturesPath+"/client.key.pem",
			),
			GinkgoWriter,
			GinkgoWriter,
		)
		Expect(err).ToNot(HaveOccurred())

		Eventually(metricExporterSession.Buffer).Should(gbytes.Say("metric-exporter.locket-lock.acquired-lock"))
		Eventually(metricExporterSession.Buffer).Should(gbytes.Say("metric-exporter.started"))
	})

	It("Should hang if it cannot acquire a lock", func() {
		mockLocketServer = mockLocket.NewMockLocketServer(fixturesPath, "neverGrantLock")
		err = mockLocketServer.Run()
		Expect(err).NotTo(HaveOccurred())

		metricExporterSession, err := gexec.Start(
			exec.Command(
				metricExporterPath,
				"--debug",
				"--enable-locking",
				"--locket-address="+mockLocketServer.ListenAddress,
				"--locket-ca-cert="+fixturesPath+"/ca.cert.pem",
				"--locket-client-cert="+fixturesPath+"/client.cert.pem",
				"--locket-client-key="+fixturesPath+"/client.key.pem",
			),
			GinkgoWriter,
			GinkgoWriter,
		)
		Expect(err).ToNot(HaveOccurred())

		Eventually(metricExporterSession.Buffer).Should(gbytes.Say("metric-exporter.locket-lock.failed-to-acquire-lock"))
		Consistently(metricExporterSession.Buffer, 5*time.Second).ShouldNot(gbytes.Say("metric-exporter.started"))
	})

	It("Should hang until it acquires a lock, then start", func() {
		mockLocketServer = mockLocket.NewMockLocketServer(fixturesPath, "grantLockAfterFiveAttempts")
		err = mockLocketServer.Run()
		Expect(err).NotTo(HaveOccurred())

		metricExporterSession, err := gexec.Start(
			exec.Command(
				metricExporterPath,
				"--debug",
				"--enable-locking",
				"--locket-address="+mockLocketServer.ListenAddress,
				"--locket-ca-cert="+fixturesPath+"/ca.cert.pem",
				"--locket-client-cert="+fixturesPath+"/client.cert.pem",
				"--locket-client-key="+fixturesPath+"/client.key.pem",
			),
			GinkgoWriter,
			GinkgoWriter,
		)
		Expect(err).ToNot(HaveOccurred())

		Eventually(metricExporterSession.Buffer).Should(gbytes.Say("metric-exporter.locket-lock.failed-to-acquire-lock"))
		// By default locketClient retries every one second with a TTL of 15 seconds.
		// The mock server is set to release the lock after 5 attempts, so we need to wait more than 5 seconds
		Eventually(metricExporterSession.Buffer, 10*time.Second).Should(gbytes.Say("metric-exporter.locket-lock.acquired-lock"))
		Eventually(metricExporterSession.Buffer).Should(gbytes.Say("metric-exporter.started"))
	})

	It("Should crash if it loses the lock", func() {
		mockLocketServer = mockLocket.NewMockLocketServer(fixturesPath, "grantLockOnceThenFail")
		err = mockLocketServer.Run()
		Expect(err).NotTo(HaveOccurred())

		metricExporterSession, err := gexec.Start(
			exec.Command(
				metricExporterPath,
				"--debug",
				"--enable-locking",
				"--locket-address="+mockLocketServer.ListenAddress,
				"--locket-ca-cert="+fixturesPath+"/ca.cert.pem",
				"--locket-client-cert="+fixturesPath+"/client.cert.pem",
				"--locket-client-key="+fixturesPath+"/client.key.pem",
			),
			GinkgoWriter,
			GinkgoWriter,
		)
		Expect(err).ToNot(HaveOccurred())

		Eventually(metricExporterSession.Buffer).Should(gbytes.Say("metric-exporter.locket-lock.acquired-lock"))
		Eventually(metricExporterSession.Buffer, 10*time.Second).Should(gbytes.Say("metric-exporter.locket-lock.lost-lock"))
		Eventually(metricExporterSession.Buffer, 30*time.Second).Should(gbytes.Say("metric-exporter.process-group-stopped-with-error"))
		Eventually(metricExporterSession, 30*time.Second).Should(gexec.Exit())
	})
})
