generate:
	cd schema && protoc --go_out=plugins=grpc:../bouncer bouncer.proto

build:
	go build -o .bin/bouncer ./cmd/main.go

run:
	docker-compose -f ./docker-compose.yaml up -d --build

down:
	docker-compose -f ./docker-compose.yaml down

test:
	go test -v -race -count 100 ./bouncer/...

itests:
	docker-compose -f ./docker-compose-tests.yaml up --build --abort-on-container-exit --exit-code-from itests && \
	docker-compose -f ./docker-compose-tests.yaml down