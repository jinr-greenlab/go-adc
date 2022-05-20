.PHONY: build docker-build

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
	docker tag go-adc:latest ${DOCKER_IMAGE}:latest
#	docker tag go-adc:latest ${DOCKER_IMAGE}:${COMMIT}-${TIMESTAMP}
#	@echo docker push ${DOCKER_IMAGE}:${COMMIT}-${TIMESTAMP}
	@echo docker push ${DOCKER_IMAGE}:latest
