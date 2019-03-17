#!/bin/bahs
export CGO_ENABLED=0 
export GOOS=linux 
export GOARCH=amd64 
go build -o build/UploadRysnc src/opslabgo/main.go