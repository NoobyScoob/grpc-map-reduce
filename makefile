BINARY=bin/main

build:
	GOARCH=amd64 GOOS=darwin go build -o ${BINARY}_darwin main.go
	GOARCH=amd64 GOOS=linux go build -o ${BINARY}_linux main.go

run: build
	echo "Running base example word count"
	./bin/main_darwin client ./input/small/ wc
	killall main

test: testwc testii

testwc:
	echo "Running word count on large input"
	./bin/main_darwin client ./input/large/ wc
	killall main

testii:
	echo "Running inverted index on large input"
	./bin/main_darwin client ./input/large/ ii
	killall main

clean:
	rm bin/*