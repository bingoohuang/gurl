package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/bingoohuang/gg/pkg/man"

	"go.uber.org/atomic"

	"github.com/bingoohuang/gg/pkg/fla9"
)

var (
	disableKeepAlive, ver, form, pretty, ugly, raw, insecureSSL, gzipOn, isjson bool
	auth, proxy, printV, body, think, caFile, download, method                  string

	uploadFiles, urls []string
	printOption       uint8
	benchN, benchC    int
	currentN          atomic.Int64
	timeout           time.Duration
	limitRate         = NewRateLimitFlag()

	jsonmap = map[string]interface{}{}
)

func init() {
	fla9.BoolVar(&isjson, "json,j", true, "Send the data as a JSON object")
	flagEnv(&urls, "url,u", "", "HTTP request URL")
	fla9.StringVar(&method, "method,m", "GET", "HTTP method")

	fla9.BoolVar(&disableKeepAlive, "k", false, "Disable Keepalive enabled")
	fla9.BoolVar(&ver, "version,v", false, "Print Version Number")
	fla9.BoolVar(&raw, "raw,r", false, "Print JSON Raw Format")
	fla9.BoolVar(&ugly, "ugly", false, "Print JSON In Ugly compact Format")
	fla9.StringVar(&printV, "print,p", "A", "Print request and response")
	fla9.StringVar(&caFile, "ca", "", "ca certificate file")
	fla9.BoolVar(&form, "f", false, "Submitting as a form")
	fla9.BoolVar(&gzipOn, "gzip", false, "Gzip request body or not")
	fla9.StringVar(&download, "d", "", "Download the url content as file, yes/no")
	fla9.BoolVar(&insecureSSL, "i", false, "Allow connections to SSL sites without certs")
	fla9.DurationVar(&timeout, "t", 1*time.Minute, "Timeout for read and write")
	fla9.StringsVar(&uploadFiles, "F", nil, "Upload files")
	fla9.Var(limitRate, "L", "Limit rate /s, append :req/:rsp to specific the limit direction")
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
	printVerbose
	printHTTPTrace
)

func parsePrintOption(s string) {
	AdjustPrintOption(&s, 'A', printReqHeader|printReqBody|printRespHeader|printRespBody|printReqSession)
	AdjustPrintOption(&s, 'a', printReqHeader|printReqBody|printRespHeader|printRespBody|printReqSession)
	AdjustPrintOption(&s, 'H', printReqHeader)
	AdjustPrintOption(&s, 'B', printReqBody)
	AdjustPrintOption(&s, 'h', printRespHeader)
	AdjustPrintOption(&s, 'b', printRespBody)
	AdjustPrintOption(&s, 's', printReqSession)
	AdjustPrintOption(&s, 'v', printVerbose)
	AdjustPrintOption(&s, 't', printHTTPTrace)

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
  -d                Download the url content as file
  -F filename       Upload a file, e.g. gurl :2110 -F 1.png -F 2.png
  -L limit          Limit rate for upload or download /s, like 10K
  -raw,r            Print JSON Raw format other than pretty
  -i                Allow connections to SSL sites without certs
  -j                Send the data in a JSON object as application/json
  -ca               Ca certificate file
  -proxy=PROXY_URL  Proxy with host and port
  -print,p          String specifying what the output should contain, default will print all information
                       H: request headers  B: request body  h: response headers  b: response body s: http conn session v: Verbose t: HTTP trace
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

type RateLimitDirection int

const (
	RateLimitBoth RateLimitDirection = iota
	RateLimitRequest
	RateLimitResponse
)

func NewRateLimitFlag() *RateLimitFlag {
	val := uint64(0)
	return &RateLimitFlag{Val: &val}
}

type RateLimitFlag struct {
	Val       *uint64
	Direction RateLimitDirection
}

func (i *RateLimitFlag) Enabled() bool { return i.Val != nil && *i.Val > 0 }

func (i *RateLimitFlag) String() string {
	if !i.Enabled() {
		return "0"
	}

	s := man.Bytes(*i.Val)
	switch i.Direction {
	case RateLimitRequest:
		return s + ":req"
	case RateLimitResponse:
		return s + ":rsp"
	}

	return s
}

func (i *RateLimitFlag) Set(value string) (err error) {
	dirPos := strings.IndexByte(value, ':')
	i.Direction = RateLimitBoth
	if dirPos > 0 {
		switch dir := value[dirPos+1:]; strings.ToLower(dir) {
		case "req":
			i.Direction = RateLimitRequest
		case "rsp":
			i.Direction = RateLimitResponse
		default:
			log.Fatalf("unknown rate limit %s", value)
		}
		value = value[:dirPos]
	}
	*i.Val, err = man.ParseBytes(value)
	return err
}

func (i *RateLimitFlag) IsForReq() bool {
	return i.Enabled() && (i.Direction == RateLimitRequest || i.Direction == RateLimitBoth)
}

func (i *RateLimitFlag) IsForRsp() bool {
	return i.Enabled() && (i.Direction == RateLimitResponse || i.Direction == RateLimitBoth)
}

func (i *RateLimitFlag) Float64() float64 { return float64(*i.Val) }
