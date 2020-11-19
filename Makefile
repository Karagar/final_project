generate:
	cd schema && protoc --go_out=plugins=grpc:../pkg bouncer.proto

build:
	go build -o .bin/bouncer ./cmd/main.go

run:
	docker-compose -f ./docker-compose.yaml up -d

down:
	docker-compose -f ./docker-compose.yaml down

test:
	go test -v -race -count 100 ./pkg/...

itests:
	docker-compose -f ./docker-compose-tests.yaml up -d --abort-on-container-exit --exit-code-from itests && \
	docker-compose -f ./docker-compose-tests.yaml down