COMMANDS := aade-invoice aade-list aade-read aade-template-from-mark aade-income aade-expenses aade-vat aade-e3

.PHONY: all clean test
all: $(addprefix bin/,$(COMMANDS))

bin/%: cmd/% $(wildcard internal/*/*.go)
	go build -o $@ ./$<

test:
	go vet ./...
	go test ./...

clean:
	rm -rf bin
