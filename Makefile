generate:
	cd schema && protoc --go_out=plugins=grpc:../pkg bouncer.proto

srv:
	go run ./service/main.go

cli:
	go run ./client/main.go

build:
	go build -o ./client ./client/main.go && go build -o ./service ./service/main.go 

test:
	go test -race -count 100 ./pkg/...

run:
	go run ./service/main.go
	# docker-compose -f ./docker-compose.yaml up -d

down:
	docker-compose -f ./docker-compose.yaml down