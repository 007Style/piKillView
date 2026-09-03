VERSION := 1.0
BINARY  := piKillView
GOFLAGS := -ldflags "-s -w"

.PHONY: build clean

build:
	CGO_ENABLED=1 go build $(GOFLAGS) -o $(BINARY) .

clean:
	rm -f $(BINARY)
