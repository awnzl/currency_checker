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
all: grpc build

.PHONY: build
build:
	$(foreach app,$(APPS),$(call BUILD,$(app));)

.PHONY: grpc
grpc:
	protoc -I ${GOBASE} \
	--go_out=paths=source_relative:${PROTO_DIR} \
	--go-grpc_out=paths=source_relative:${PROTO_DIR} \
	${PROTO_DIR}/*/*.proto

.PHONY: clean
clean:
	rm -rf $(GOBIN)
