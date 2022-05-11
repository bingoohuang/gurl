package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/bingoohuang/gg/pkg/iox"
)

func downloadFile(req *Request, res *http.Response, filename string) {
	fd, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0o666)
	if err != nil {
		log.Fatalf("create download file %q failed: %v", filename, err)
	}

	printRequestResponseForNonWindows(req, res, true)

	fmt.Printf("\nDownloading to %q\n", filename)

	total, _ := strconv.ParseInt(res.Header.Get("Content-Length"), 10, 64)
	pb := NewProgressBar(total).Start()
	br := newProgressBarReader(res.Body, pb)

	if res.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(br)
		if err != nil {
			log.Fatalf("create download file %q failed: %v", filename, err)
		}
		br = reader
	}

	// disable timeout for downloading.
	// A zero value for t means I/O operations will not time out.
	if req.ConnInfo.Conn != nil {
		if err := req.ConnInfo.Conn.SetDeadline(time.Time{}); err != nil {
			log.Printf("failed to set deadline: %v", err)
		}
	}

	if _, err := io.Copy(fd, br); err != nil {
		// A successful Copy returns err == nil, not err == EOF.
		log.Fatalf("download file %q failed: %v", filename, err)
	}
	pb.Finish()
	iox.Close(fd, br)
	fmt.Println()
}
