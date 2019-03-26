.PHONY: k8s-up k8s-down

all: install

VERSION := $(shell git describe --tags --always --dirty)
UNAME_S := $(shell uname -s)

k8s-up:
	docker-compose up -d
	docker-compose ps

k8s-down:
	docker-compose down

install_dependencies:
	dep ensure -v

install:
	go install ./cmd/...

kconfig:
ifeq ($(UNAME_S), Darwin)
	@echo "Building docker-machine config"
	@scripts/docker-machine-config.sh
else
	@echo "Building localhost config"
	@scripts/local-config.sh
endif
