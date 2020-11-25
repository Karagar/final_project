FROM golang:1.15-alpine AS build

WORKDIR $GOPATH/src/github.com/Karagar/final_project

COPY . ./
RUN go mod download
RUN go build -o .bin/bouncer ./cmd/main.go
ENTRYPOINT [".bin/bouncer"]