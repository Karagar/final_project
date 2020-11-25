FROM golang:1.15-alpine AS build

WORKDIR $GOPATH/src/github.com/Karagar/final_project

COPY . ./
RUN apk --no-cache add build-base
RUN go mod download
CMD go test -v ./tests/...