APP := torrent-cli
CMD := ./tui/main.go
LDFLAGS := -s -w -X main.Version=$(VERSION)

.PHONY: all build run test bench race fmt vet clean

all: fmt vet test build

build:
	go build -trimpath -ldflags="$(LDFLAGS)" -o bin/$(APP) $(CMD)

# GOAMD64=v3 go build -ldflags="-s -w" -o main main.go
build-amd:
	GOAMD64=v3 go build -ldflags="$(LDFLAGS)" -o bin/$(APP) $(CMD)

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
