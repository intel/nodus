.PHONY: k8s-up k8s-down kconfig

all: install

VERSION := $(shell git describe --tags --always --dirty)
UNAME_S := $(shell uname -s)

K8S_SOURCE := $(or $(K8S_SOURCE), ${GOPATH}/src/github.com/k8s.io/kubernetes/)

SCHEDULER_VERBOSITY := $(or $(SCHEDULER_VERBOSITY), 5)

APISERVER := $(or $(APISERVER), localhost:8080)

# controller-manager is not run by default as it marks nodes unreachable. To run it with the controller manager use:
# DOCKER_COMPOSE_SERVICES="etcd k8s-api k8s-scheduler k8s-controller-manager" k8s-up
DOCKER_COMPOSE_SERVICES := $(or $(DOCKER_COMPOSE_SERVICES), etcd k8s-api k8s-scheduler)

k8s-up:
	docker-compose up -d ${DOCKER_COMPOSE_SERVICES}
	docker-compose ps

k8s-down:
	docker-compose down

install_dependencies:
	dep ensure

test:
	go test -v ./pkg/...

install: test
	go install ./cmd/...

scheduler-build:
	@cd ${K8S_SOURCE} && make WHAT=cmd/kube-scheduler GOGCFLAGS="-N -l -v"
	@cp ${K8S_SOURCE}/_output/bin/kube-scheduler .

scheduler-run:
	./kube-scheduler --master=${APISERVER} -v ${SCHEDULER_VERBOSITY}

kconfig:
ifdef DOCKER_MACHINE_NAME
	@echo "Building docker-machine config"
	@scripts/docker-machine-config.sh
else
	@echo "Building localhost config"
	@scripts/local-config.sh
endif
