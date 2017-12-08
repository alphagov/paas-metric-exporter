.DEFAULT_GOAL := help

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'


.PHONY: test
test: unit ## Run tests

.PHONY: unit
unit: $(GOPATH)/bin/ginkgo ## Run unit tests
	ginkgo -r

$(GOPATH)/bin/ginkgo:
	cd vendor/github.com/onsi/ginkgo/ginkgo && go install .

.PHONY: generate
generate: ## Generate the test mocks
	go generate ./...
