package main

import (
	"fmt"
	"os"
)

const help = `gurl is a Go implemented CLI cURL-like tool for humans.
Usage:
	gurl [flags] [METHOD] URL [ITEM [ITEM]]
flags:
  -auth=USER[:PASS] Pass a username:password pair as the argument
  -n=0 -c=100       Number of requests and concurrency to run
  -body=""          Send RAW data as body, or @filename to load body from the file's content
  -f                Submitting the data as a form
  -F filename       Upload a file, e.g. gurl :2110 -F 1.png -F 2.png
  -L limit          Limit rate for upload or download /s, like 10K
  -j                Send the data in a JSON object as application/json
  -raw              Print JSON Raw format other than pretty
  -i                Allow connections to SSL sites without certs
  -ca               Ca certificate file
  -proxy=PROXY_URL  Proxy with host and port
  -print=A          String specifying what the output should contain, default will print all information
                       H: request headers  B: request body  h: response headers  b: response body s: http conn session
  -t                Set timeout for read and write, default 1m
  -k                Disable keepalive
  -think            Think time, like 5s, 100ms, 100ms-5s, 100-200ms and etc.
  -v                Show Version Number
METHOD:
  gurl defaults to either GET (if there is no request data) or POST (with request data).
URL:
  The only one needed to perform a request is a URL. The default scheme is http://,
  which can be omitted from the argument; example.org works just fine.
ITEM:
  Can be any of: Query      : key=value  Header: key:value       Post data: key=value 
                 Force query: key==value key==@/path/file
                 JSON data  : key:=value Upload: key@/path/file
                 File content as body: @/path/file
Example:
  gurl beego.me
more help information please refer to https://github.com/bingoohuang/gurl
`

func usage() {
	fmt.Print(help)
	os.Exit(2)
}
