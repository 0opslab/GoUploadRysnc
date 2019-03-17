export CGO_ENABLED=0
export GOOS=darwin
export GOARCH=amd64
go build -o build/UploadRysnc src/opslabgo/main.go