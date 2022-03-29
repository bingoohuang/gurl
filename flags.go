package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/bingoohuang/gg/pkg/fla9"
)

var (
	disableKeepAlive, ver, form, pretty, raw, download, insecureSSL bool
	auth, proxy, printV, body, think, caFile                        string

	uploadFiles    []string
	printOption    uint8
	benchN, benchC int
	timeout        time.Duration
	limitRate      uint64

	isjson  = fla9.Bool("json,j", true, "Send the data as a JSON object")
	method  = fla9.String("method,m", "GET", "HTTP method")
	Urls    = flagEnv("url,u", "", "HTTP request URL")
	jsonmap = map[string]interface{}{}
)

func init() {
	fla9.BoolVar(&disableKeepAlive, "k", false, "Disable Keepalive enabled")
	fla9.BoolVar(&ver, "version,v", false, "Print Version Number")
	fla9.BoolVar(&raw, "raw,r", false, "Print JSON Raw Format")
	fla9.StringVar(&printV, "print,p", "A", "Print request and response")
	fla9.StringVar(&caFile, "ca", "", "ca certificate file")
	fla9.BoolVar(&form, "f", false, "Submitting as a form")
	fla9.BoolVar(&download, "d", false, "Download the url content as file")
	fla9.BoolVar(&insecureSSL, "i", false, "Allow connections to SSL sites without certs")
	fla9.DurationVar(&timeout, "t", 1*time.Minute, "Timeout for read and write")
	fla9.StringsVar(&uploadFiles, "F", nil, "Upload files")
	fla9.SizeVar(&limitRate, "L", "0", "Limit rate /s")
	fla9.StringVar(&think, "think", "0", "Think time")

	flagEnvVar(&auth, "auth", "", "HTTP authentication username:password, USER[:PASS]")
	flagEnvVar(&proxy, "proxy,P", "", "Proxy host and port, PROXY_URL")
	fla9.IntVar(&benchN, "n", 1, "Number of bench requests to run")
	fla9.IntVar(&benchC, "c", 1, "Number of bench requests to run concurrently.")
	fla9.StringVar(&body, "body,b", "", "Raw data send as body")
}

const (
	printReqHeader uint8 = 1 << iota
	printReqBody
	printRespHeader
	printRespBody
	printReqSession
)

func parsePrintOption(s string) {
	AdjustPrintOption(&s, 'A', printReqHeader|printReqBody|printRespHeader|printRespBody|printReqSession)
	AdjustPrintOption(&s, 'a', printReqHeader|printReqBody|printRespHeader|printRespBody|printReqSession)
	AdjustPrintOption(&s, 'H', printReqHeader)
	AdjustPrintOption(&s, 'B', printReqBody)
	AdjustPrintOption(&s, 'h', printRespHeader)
	AdjustPrintOption(&s, 'b', printRespBody)
	AdjustPrintOption(&s, 's', printReqSession)

	if s != "" {
		log.Fatalf("unknown print option: %s", s)
	}
}

func AdjustPrintOption(s *string, r rune, flags uint8) {
	if strings.ContainsRune(*s, r) {
		printOption |= flags
		*s = strings.ReplaceAll(*s, string(r), "")
	}
}

func HasPrintOption(flags uint8) bool {
	return printOption&flags == flags
}

const help = `gurl is a Go implemented cURL-like cli tool for humans.
Usage:
	gurl [flags] [METHOD] URL [URL] [ITEM [ITEM]]
flags:
  -auth=USER[:PASS] Pass a username:password pair as the argument
  -n=0 -c=100       Number of requests and concurrency to run
  -body,b           Send RAW data as body, or @filename to load body from the file's content
  -f                Submitting the data as a form
  -F filename       Upload a file, e.g. gurl :2110 -F 1.png -F 2.png
  -L limit          Limit rate for upload or download /s, like 10K
  -raw,r            Print JSON Raw format other than pretty
  -i                Allow connections to SSL sites without certs
  -j                Send the data in a JSON object as application/json
  -ca               Ca certificate file
  -proxy=PROXY_URL  Proxy with host and port
  -print,p          String specifying what the output should contain, default will print all information
                       H: request headers  B: request body  h: response headers  b: response body s: http conn session
  -t                Set timeout for read and write, default 1m
  -k                Disable keepalive
  -think            Think time, like 5s, 100ms, 100ms-5s, 100-200ms and etc.
  -version,v        Show Version Number
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
  gurl :8080
more help information please refer to https://github.com/bingoohuang/gurl
`

func usage() {
	fmt.Print(help)
	os.Exit(2)
}
