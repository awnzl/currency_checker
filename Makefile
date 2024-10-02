GOBUILD   := go build

GOBASE    := $(shell pwd)
CMD       := $(GOBASE)/cmd
GOBIN     := $(GOBASE)/bin
PROTO_DIR := $(GOBASE)/lib/proto
PACKAGE   := $(shell head -1 go.mod | awk '{print $$2}')

APPS      := $(notdir $(wildcard $(CMD)/*))

define BUILD
$(GOBUILD) -o $(GOBIN)/$(1) $(CMD)/$(1)/*.go
endef

.PHONY: all
all: docker

.PHONY: local
local: proto build

.PHONY: build
build:
	$(foreach app,$(APPS),$(call BUILD,$(app));)

.PHONY: proto
proto:
	protoc -I ${GOBASE} \
	--go_out=paths=source_relative:${GOBASE} \
	--go-grpc_out=paths=source_relative:${GOBASE} \
	${PROTO_DIR}/*/*.proto

.PHONY: docker
make docker:
	./dockers/build.sh

.PHONY: clean
clean:
	rm -rf $(GOBIN)
	rm -rf ${PROTO_DIR}/*/*.go
	rm -rf ./dockers/build.log
