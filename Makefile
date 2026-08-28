.PHONY: build test dist clean

build:
	CGO_ENABLED=0 go build -o aichange .

test:
	CGO_ENABLED=0 go test ./...

dist:
	mkdir -p dist
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dist/aichange-linux-amd64 .
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o dist/aichange-linux-arm64 .
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o dist/aichange-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o dist/aichange-darwin-arm64 .
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o dist/aichange-windows-amd64.exe .
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -o dist/aichange-windows-arm64.exe .

clean:
	rm -f aichange aichange.exe
	rm -rf dist
