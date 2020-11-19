generate:
	cd schema && protoc --go_out=plugins=grpc:../pkg bouncer.proto

build:
	go build -o .bin/bouncer ./cmd/main.go

test:
	go test -v -race -count 100 ./pkg/...

itest:
	docker-compose -f ./docker-compose-test.yaml up -d

run:
	go run ./cmd/main.go
	# docker-compose -f ./docker-compose.yaml up -d

down:
	docker-compose -f ./docker-compose.yaml down