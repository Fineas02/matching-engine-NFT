build:
	go build -o bin/exchange

run: build
	./bin/exchange

test:
	go test -v ./...

deadlock-test:
	GODEBUG=sync=1 go test -race -v -run TestPlaceAndFillOrdersConcurrently ./orderbook
