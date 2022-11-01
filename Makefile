.PHONY: build docker-build swagger

build:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w' -o bin/go-adc

swagger:
	swagger generate spec -o swaggerui/swagger.json

DOCKER_IMAGE := quay.io/kozhukalov/go-adc
TIMESTAMP ?= $(shell date +%Y%m%d%H%M%S)
COMMIT    ?= $(shell git rev-parse HEAD)

docker-build:
	docker build --rm -t go-adc:latest .

docker-pub:
	docker tag go-adc:latest ${DOCKER_IMAGE}:tqdc
#	docker tag go-adc:latest ${DOCKER_IMAGE}:${COMMIT}-${TIMESTAMP}
#	@echo docker push ${DOCKER_IMAGE}:${COMMIT}-${TIMESTAMP}
	@echo docker push ${DOCKER_IMAGE}:tqdc

# linting
LINTER              := golangci-lint
LINTER_CONFIG       := .golangci.yaml

# Our primary linter (golangci-lint) uses an embedded variant golint. This
# embedded version will catch the most egregious of the issues that the
# standard golint will catch, but it will fail to catch missing documentation.
# The purpose of this script is to produce a nonzero return code if the
# standard golint detects any issues.
.PHONY: lint
lint:
lint:
	$(LINTER) run --config $(LINTER_CONFIG)
	@git --no-pager show --check
#	@./tools/golint
