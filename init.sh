#!/bin/bash

go get golang.org/dl/go1.20.1@latest
if ! [[ ":$PATH:" == *":$HOME/go/bin:"* ]]; then
  export PATH="$PATH:$HOME/go/bin"
fi


if ! command -v protoc &> /dev/null; then
  if [[ ":$PATH:" == *":$HOME/bin:"* ]]; then
    PB_REL="https://github.com/protocolbuffers/protobuf/releases"
    FILE_NAME="protoc-3.15.8-linux-x86_64.zip"
    curl -LO $PB_REL/download/v3.15.8/$FILE_NAME
    unzip protoc-3.15.8-linux-x86_64.zip -d ./protoc
    cp ./protoc/bin/* $HOME/bin/
    rm -rf protoc $FILE_NAME
  else
    echo "not testing on luddy server"
  fi
else
  echo "protoc compiler already exists!"
fi

# protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative services/reducer.proto 
go run main.go client wordcount nMappers=3 nReducers=1 fn=wc
go run main.go mapper port