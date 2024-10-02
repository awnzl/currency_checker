FROM golang:1.23.1-alpine AS builder
WORKDIR /app_src

RUN apk update && apk add --no-cache make protobuf-dev

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest \
    && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

COPY ../cmd ./cmd
COPY ../lib ./lib
COPY ../go.mod ./go.mod
COPY ../go.sum ./go.sum
COPY ../Makefile ./Makefile

RUN go mod download && go mod verify
RUN make local
