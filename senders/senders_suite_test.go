package senders_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSenders(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Senders Suite")
}
