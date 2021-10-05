all:
	@echo "try: make test"

test: lint
	go test -race -covermode=atomic ./...
	# Test 32 bit OSes.
	GOOS=linux GOARCH=386 go build .
	GOOS=freebsd GOARCH=386 go build .

lint:
	# Test lint on four platforms.
	GOOS=linux golangci-lint run --enable-all -D maligned,scopelint,interfacer,golint,tagliatelle,exhaustivestruct,cyclop
	GOOS=darwin golangci-lint run --enable-all -D maligned,scopelint,interfacer,golint,tagliatelle,exhaustivestruct,cyclop
	GOOS=windows golangci-lint run --enable-all -D maligned,scopelint,interfacer,golint,tagliatelle,exhaustivestruct,cyclop
	GOOS=freebsd golangci-lint run --enable-all -D maligned,scopelint,interfacer,golint,tagliatelle,exhaustivestruct,cyclop
