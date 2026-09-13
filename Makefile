COMMANDS := aade-invoice aade-list aade-read

.PHONY: all clean test
all: $(addprefix bin/,$(COMMANDS))

bin/%: cmd/% $(wildcard internal/*/*.go)
	go build -o $@ ./$<

test:
	go vet ./...
	go test ./...

clean:
	rm -rf bin
