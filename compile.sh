cd workflow
GOOS=darwin GOARCH=amd64 go build -o arxiv_amd64 arxiv.go
GOOS=darwin GOARCH=arm64 go build -o arxiv_arm64 arxiv.go
cd ../download
GOOS=darwin GOARCH=amd64 go build -o download_amd64 download.go
GOOS=darwin GOARCH=arm64 go build -o download_arm64 download.go
cd ..