package presenters_test

import (
	. "github.com/alphagov/paas-metric-exporter/presenters"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type MyStruct struct {
	Foo string
	Bar string
}

var _ = Describe("PathPresenter", func() {
	Describe("#Present", func() {
		It("should present the data according to the template", func() {
			presenter, _ := NewPathPresenter("{{.Foo}}-{{.Bar}}")
			data := MyStruct{Foo: "foo", Bar: "bar"}
			output, err := presenter.Present(data)

			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(Equal("foo-bar"))
		})

		It("should fail to construct the presenter due to lack of dot", func() {
			_, err := NewPathPresenter("{{Foo}}")
			Expect(err).Should(MatchError("template: metric:1: function \"Foo\" not defined"))
		})

		It("should fail to present the data due to unknown property in template", func() {
			presenter, _ := NewPathPresenter("{{.Missing}}")
			data := MyStruct{Foo: "foo", Bar: "bar"}
			_, err := presenter.Present(data)

			Expect(err).Should(MatchError("template: metric:1:2: executing \"metric\" at <.Missing>: can't evaluate field Missing in type presenters_test.MyStruct"))
		})
	})
})
