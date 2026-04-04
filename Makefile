.PHONY: build test run clean

build:
	go build -o bin/fauxtes ./cmd/fauxtes

test:
	go test ./... -v -count=1

run: build
	./bin/fauxtes

clean:
	rm -rf bin/
