COMMIT_ID := $(shell git rev-parse HEAD)

build:
	go build -ldflags "-X main.version=$(COMMIT_ID)"

clean:
	rm search-logs
