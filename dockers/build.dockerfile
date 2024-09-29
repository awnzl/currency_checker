FROM golang:1.23.1-alpine AS builder
WORKDIR /app_src

RUN apk add --no-cache make
RUN go mod download && go mod verify
RUN make all
