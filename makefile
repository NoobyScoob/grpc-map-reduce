BINARY=bin/server

build:
	chmod +x init.sh
	./init.sh
	GOARCH=amd64 GOOS=darwin go1.20.1 build -o ${BINARY}_darwin main.go
	GOARCH=amd64 GOOS=linux go1.20.1 build -o ${BINARY}_linux main.go

run: build
	echo "Running base example word count"
	./bin/server_linux client ./input/small/ wc
	killall main

test: testwc testii

testwc:
	echo "Running word count on large input"
	./bin/server_linux client ./input/large/ wc
	killall main

testii:
	echo "Running inverted index on large input"
	./bin/server_linux client ./input/large/ ii
	killall main

clean:
	rm bin/*