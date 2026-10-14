.PHONY: build test vet

build:
	cd cli-go && go build -o ../bin/whatsapp .

test:
	cd cli-go && go test ./...

vet:
	cd cli-go && go vet ./... && test -z "$$(gofmt -l .)"
