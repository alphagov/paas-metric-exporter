package presenters_test

import (
	. "github.com/alphagov/paas-metric-exporter/presenters"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SnakeCasePresenter", func() {
	presenter := NewSnakeCasePresenter()

	Describe("#Present", func() {
		It("snake cases camel case strings", func() {
			result := presenter.Present("fooBar")
			Expect(result).To(Equal("foo_bar"))
		})

		It("snake cases upper camel case strings", func() {
			result := presenter.Present("FooBar")
			Expect(result).To(Equal("foo_bar"))
		})

		It("lower cases acronyms rather than snake casing each letter", func() {
			result := presenter.Present("GUID")
			Expect(result).To(Equal("guid"))

			result = presenter.Present("AppID")
			Expect(result).To(Equal("app_id"))
		})
	})
})
