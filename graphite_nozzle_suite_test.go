package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestGraphiteNozzle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GraphiteNozzle Suite")
}
