set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64
go build -o build/UploadRysnc.exe src/opslabgo/main.go