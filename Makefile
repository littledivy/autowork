.PHONY: build fmt check clean install

build:
	go build -o autowork .

fmt:
	go fmt ./...

check:
	go vet ./...

clean:
	rm -f autowork

install: build
	cp autowork $(GOPATH)/bin/ 2>/dev/null || cp autowork ~/go/bin/
