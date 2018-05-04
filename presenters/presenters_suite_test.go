package presenters_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPresenters(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Presenters Suite")
}
