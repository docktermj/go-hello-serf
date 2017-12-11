# Makefile that builds go-hello-serf, a "go" program.

# PROGRAM_NAME is the name of the GIT repository.
PROGRAM_NAME := $(shell basename `git rev-parse --show-toplevel`)
TARGET_DIRECTORY := ./target
DOCKER_CONTAINER_NAME := $(PROGRAM_NAME)
DOCKER_IMAGE_NAME := local/$(PROGRAM_NAME)
BUILD_VERSION := $(shell git describe --always --tags --abbrev=0 --dirty)
BUILD_TAG := $(shell git describe --always --tags --abbrev=0)
BUILD_ITERATION := $(shell git log $(BUILD_TAG)..HEAD --oneline | wc -l | sed -e 's/^[ \t]*//')


# The first "make" target runs as default.
.PHONY: default
default: build-local

# -----------------------------------------------------------------------------
# Local development
# -----------------------------------------------------------------------------

.PHONY: build-local
build-local:
	go install github.com/docktermj/$(PROGRAM_NAME)


.PHONY: test-local
test-local:
	go test github.com/docktermj/$(PROGRAM_NAME)/... 

# -----------------------------------------------------------------------------
# Docker-based development
# -----------------------------------------------------------------------------

.PHONY: build
build: docker-build
	mkdir -p $(TARGET_DIRECTORY) || true
	docker rm --force $(DOCKER_CONTAINER_NAME) || true
	docker create \
		--name $(DOCKER_CONTAINER_NAME) \
		$(DOCKER_IMAGE_NAME)
	docker cp $(DOCKER_CONTAINER_NAME):/output/. $(TARGET_DIRECTORY) || true
	docker rm --force $(DOCKER_CONTAINER_NAME)


.PHONY: docker-build
docker-build:
	docker build \
		--build-arg PROGRAM_NAME=$(PROGRAM_NAME) \
		--build-arg BUILD_VERSION=$(BUILD_VERSION) \
		--build-arg BUILD_ITERATION=$(BUILD_ITERATION) \
		--tag $(DOCKER_IMAGE_NAME) \
		.


# -----------------------------------------------------------------------------
# Demo targets
# -----------------------------------------------------------------------------

.PHONY: build-demo
build-demo: build-local
	mkdir -p $(TARGET_DIRECTORY) || true
	cp ${GOPATH}/bin//$(PROGRAM_NAME) $(TARGET_DIRECTORY)
	docker build \
		--tag $(DOCKER_IMAGE_NAME)-demo \
		--file demo.Dockerfile \
		.


# -----------------------------------------------------------------------------
# Utility targets
# -----------------------------------------------------------------------------

.PHONY: docker-run
docker-run:
	docker run \
	    --interactive \
	    --tty \
	    --name $(DOCKER_CONTAINER_NAME) \
	    $(DOCKER_IMAGE_NAME)


.PHONY: dependencies
dependencies:
	go get -u github.com/jstemmer/go-junit-report
	go get -u github.com/hashicorp/serf/client
	go get -u github.com/gorilla/mux
	go get -u github.com/hashicorp/serf/serf
	go get -u github.com/pkg/errors
	go get -u golang.org/x/sync/errgroup	
	

.PHONY: clean
clean:
	docker rm --force $(DOCKER_CONTAINER_NAME) || true
	rm -rf $(TARGET_DIRECTORY)


.PHONY: help
help:
	@echo "Build $(PROGRAM_NAME) version $(BUILD_VERSION)-$(BUILD_ITERATION)".
	@echo "All targets:"
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | xargs
