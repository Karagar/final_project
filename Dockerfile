FROM golang:1.15

WORKDIR $GOPATH/src/github.com/Karagar/final_project

COPY . ./
RUN go mod download
RUN go build -o /go/bin/bouncer ./cmd/main.go

FROM alpine:3.10
RUN apk --no-cache add ca-certificates
COPY --from=build /go/bin/bouncer /bin/

ENTRYPOINT ["bouncer"]