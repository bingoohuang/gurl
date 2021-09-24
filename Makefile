default:
	go install -trimpath -ldflags='-extldflags=-static -s -w' ./...
	upx ~/go/bin/gurl
	ls -lh ~/go/bin/gurl

linux:
	GOOS=linux GOARCH=amd64 go install -trimpath -ldflags='-extldflags=-static -s -w' ./...
	upx ~/go/bin/linux_amd64/gurl
	ls -lh ~/go/bin/linux_amd64/gurl
	bssh scp ~/go/bin/linux_amd64/gurl r:/usr/local/bin/