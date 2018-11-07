package webhook_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestWebHook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "WebHook Suite")
}
