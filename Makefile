APP := torrent-cli
CMD := ./cmd/main

.PHONY: all build run test bench race fmt vet clean

all: fmt vet test build

build:
	go build -trimpath -o bin/$(APP) $(CMD)

run:
	go run $(CMD)

test:
	go test ./...

race:
	go test -race ./...

bench:
	go test -bench=. -benchmem ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

clean:
	rm -rf bin/

profile:
	go test -cpuprofile=cpu.out -memprofile=mem.out ./...
