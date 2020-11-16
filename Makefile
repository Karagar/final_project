generate:
	cd schema && protoc --go_out=plugins=grpc:../pkg bouncer.proto

srv:
	go run ./service/main.go

build:
	go build -o ./service ./service/main.go 

test:
	go test -v -race -count 100 ./tests/...

run:
	go run ./service/main.go
	# docker-compose -f ./docker-compose.yaml up -d

down:
	docker-compose -f ./docker-compose.yaml down