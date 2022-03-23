package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/bingoohuang/gg/pkg/iox"

	"github.com/bingoohuang/goup"
	"github.com/bingoohuang/goup/shapeio"

	"github.com/bingoohuang/gg/pkg/ss"
	"github.com/bingoohuang/gg/pkg/thinktime"
	"github.com/bingoohuang/gg/pkg/v"

	"github.com/bingoohuang/gg/pkg/fla9"
)

const (
	printReqHeader uint8 = 1 << iota
	printReqBody
	printRespHeader
	printRespBody
	printReqSession
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
	flagEnvVar(&proxy, "proxy", "", "Proxy host and port, PROXY_URL")
	fla9.IntVar(&benchN, "n", 1, "Number of bench requests to run")
	fla9.IntVar(&benchC, "c", 1, "Number of bench requests to run concurrently.")
	fla9.StringVar(&body, "body,b", "", "Raw data send as body")
}

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

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	fla9.Usage = usage

	if err := fla9.CommandLine.Parse(os.Args[1:]); err != nil {
		log.Fatalf("failed to parse args, %v", err)
	}

	pretty = !raw
	nonFlagArgs := filter(fla9.Args())

	if ver {
		fmt.Println(v.Version())
		os.Exit(2)
	}

	parsePrintOption(printV)
	if !HasPrintOption(printReqBody) {
		defaultSetting.DumpBody = false
	}

	stdin := parseStdin()

	if len(*Urls) == 0 {
		log.Fatal("Miss the URL")
	}

	for _, urlAddr := range *Urls {
		run(urlAddr, nonFlagArgs, stdin)
	}
}

func parseStdin() []byte {
	if runtime.GOOS == "windows" {
		return nil
	}

	fi, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}

	var stdin []byte
	if fi.Size() != 0 {
		if stdin, err = ioutil.ReadAll(os.Stdin); err != nil {
			log.Fatal("Read from Stdin", err)
		}
	}

	return stdin
}

var uploadFilePb *ProgressBar

func run(urlAddr string, nonFlagArgs []string, stdin []byte) {
	u := parseURL(caFile != "", urlAddr)
	urlAddr = u.String()
	req := getHTTP(*method, urlAddr, nonFlagArgs, timeout)
	if u.User != nil {
		password, _ := u.User.Password()
		req.SetBasicAuth(u.User.Username(), password)
	}

	// Setup HTTPS client
	req.SetTLSClientConfig(createTlsConfig())

	// Proxy Support
	if proxy != "" {
		purl, err := url.Parse(proxy)
		if err != nil {
			log.Fatal("Proxy Url parse err", err)
		}
		req.SetProxy(http.ProxyURL(purl))
	} else {
		eurl, err := http.ProxyFromEnvironment(req.Req)
		if err != nil {
			log.Fatal("Environment Proxy Url parse err", err)
		}
		req.SetProxy(http.ProxyURL(eurl))
	}

	if len(uploadFiles) > 0 {
		var fileReaders []io.ReadCloser
		for _, uploadFile := range uploadFiles {
			fileReader, err := goup.CreateChunkReader(uploadFile, 0, 0, 0)
			if err != nil {
				log.Fatal(err)
			}
			fileReaders = append(fileReaders, fileReader)
		}

		uploadFilePb = NewProgressBar(0)
		fields := map[string]interface{}{}
		if len(fileReaders) == 1 {
			fields["file"] = fileReaders[0]
		} else {
			for i, r := range fileReaders {
				name := fmt.Sprintf("file-%d", i+1)
				fields[name] = r
			}
		}

		up := goup.PrepareMultipartPayload(fields)

		uploadFilePb.SetTotal(up.Size)

		if limitRate > 0 {
			up.Body = shapeio.NewReader(up.Body, shapeio.WithRateLimit(float64(limitRate)))
		}

		pb := &goup.PbReader{Reader: up.Body, Adder: goup.AdderFn(func(value uint64) {
			uploadFilePb.Add64(int64(value))
		})}

		req.BodyAndSize(pb, up.Size)
		req.Setting.DumpBody = false

		for hk, hv := range up.Headers {
			req.Header(hk, hv)
		}
	} else if body != "" {
		req.Body(body)
	}
	if len(stdin) > 0 {
		var j interface{}
		d := json.NewDecoder(bytes.NewReader(stdin))
		d.UseNumber()
		if err := d.Decode(&j); err != nil {
			req.Body(stdin)
		} else {
			_, _ = req.JsonBody(j)
		}
	}

	thinker, _ := thinktime.ParseThinkTime(think)
	thinkerFn := func() {}
	if thinker != nil {
		thinkerFn = func() {
			thinker.Think(true)
		}
	}

	req.SetupTransport()

	// AB bench
	if benchC > 1 {
		req.Debug(false)
		RunBench(req, thinkerFn)
		return
	}

	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			req.ConnInfo = info
		},
	}

	req.Req = req.Req.WithContext(httptrace.WithClientTrace(req.Req.Context(), trace))

	for i := 0; i < benchN; i++ {
		doRequest(req, u)
		req.Reset()
		if i < benchN-1 {
			thinkerFn()
		}
	}
}

func createTlsConfig() (tlsConfig *tls.Config) {
	if caFile != "" {
		caCert, err := ioutil.ReadFile(caFile)
		if err != nil {
			log.Fatal(err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig = &tls.Config{RootCAs: caCertPool}
	}

	// Insecure SSL Support
	if insecureSSL {
		if tlsConfig == nil {
			tlsConfig = &tls.Config{}
		}
		tlsConfig.InsecureSkipVerify = true
	}
	return
}

func doRequest(req *Request, u *url.URL) {
	if uploadFilePb != nil {
		fmt.Printf("Uploading \"%s\"\n", strings.Join(uploadFiles, "; "))
		uploadFilePb.Set(0)
		uploadFilePb.Start()
	}

	res, err := req.Response()
	if uploadFilePb != nil {
		uploadFilePb.Finish()
		fmt.Println()
	}
	if err != nil {
		log.Fatalln("execute error:", err)
	}

	fn := ""
	if d := res.Header.Get("Content-Disposition"); d != "" {
		if _, params, _ := mime.ParseMediaType(d); params != nil {
			fn = params["filename"]
		}
	}
	ct := res.Header.Get("Content-Type")
	if download || fn != "" || !ss.ContainsFold(ct, "json", "text", "xml") {
		if *method != "HEAD" {
			if fn == "" {
				_, fn = path.Split(u.Path)
			}
			if fn != "" {
				downloadFile(req, res, fn)
				return
			}
		}
	}

	// 保证 response body 被 读取并且关闭
	_, _ = req.Bytes()

	if runtime.GOOS == "windows" {
		printResponseForWindows(req, res)
	} else {
		printResponseForNonWindows(req, res)
	}
}

func printResponseForNonWindows(req *Request, res *http.Response) {
	fi, err := os.Stdout.Stat()
	if err != nil {
		panic(err)
	}
	if fi.Mode()&os.ModeDevice == os.ModeDevice {
		var dumpHeader, dumpBody []byte
		dump := req.DumpRequest()
		dps := strings.Split(string(dump), "\n")
		for i, line := range dps {
			if len(strings.Trim(line, "\r\n ")) == 0 {
				dumpHeader = []byte(strings.Join(dps[:i], "\n"))
				dumpBody = []byte(strings.Join(dps[i:], "\n"))
				break
			}
		}

		if HasPrintOption(printReqSession) {
			info := req.ConnInfo
			c := info.Conn
			connSession := fmt.Sprintf("%s->%s (reused: %t, wasIdle: %t, idle: %s)",
				c.LocalAddr(), c.RemoteAddr(), info.Reused, info.WasIdle, info.IdleTime)
			fmt.Println(Color("Conn-Session:", Magenta), Color(connSession, Yellow))
		}
		if HasPrintOption(printReqHeader) {
			fmt.Println(ColorfulRequest(string(dumpHeader)))
		}
		if HasPrintOption(printReqBody) {
			fmt.Println(string(dumpBody))
		}
		if HasPrintOption(printRespHeader) {
			fmt.Println(Color(res.Proto, Magenta), Color(res.Status, Green))
			for k, val := range res.Header {
				fmt.Printf("%s: %s\n", Color(k, Gray), Color(strings.Join(val, " "), Cyan))
			}
			fmt.Println()
		}
		if HasPrintOption(printRespBody) {
			fmt.Println(formatResponseBody(req, pretty, true))
		}
	} else {
		body := formatResponseBody(req, pretty, false)
		if _, err := os.Stdout.WriteString(body); err != nil {
			log.Fatal(err)
		}
	}
}

func printResponseForWindows(req *Request, res *http.Response) {
	var dumpHeader, dumpBody []byte
	dump := req.DumpRequest()
	dps := strings.Split(string(dump), "\n")
	for i, line := range dps {
		if len(strings.Trim(line, "\r\n ")) == 0 {
			dumpHeader = []byte(strings.Join(dps[:i], "\n"))
			dumpBody = []byte(strings.Join(dps[i:], "\n"))
			break
		}
	}
	if HasPrintOption(printReqHeader) {
		fmt.Println(string(dumpHeader))
		fmt.Println("")
	}
	if HasPrintOption(printReqBody) {
		fmt.Println(string(dumpBody))
		fmt.Println("")
	}
	if HasPrintOption(printRespHeader) {
		fmt.Println(res.Proto, res.Status)
		for k, val := range res.Header {
			fmt.Println(k, ":", strings.Join(val, " "))
		}
		fmt.Println("")
	}
	if HasPrintOption(printRespBody) {
		fmt.Println(formatResponseBody(req, pretty, false))
	}
}

func parseURL(hasCaFile bool, urls string) *url.URL {
	if urls == "" {
		usage()
	}

	schema := "http"
	if hasCaFile {
		schema = "https"
	}

	if strings.HasPrefix(urls, ":") {
		urlb := []byte(urls)
		if urls == ":" {
			urls = schema + "://localhost/"
		} else if len(urls) > 1 && urlb[1] != '/' {
			urls = schema + "://localhost" + urls
		} else {
			urls = schema + "://localhost" + string(urlb[1:])
		}
	}
	if !strings.HasPrefix(urls, "http://") && !strings.HasPrefix(urls, "https://") {
		urls = schema + "://" + urls
	}
	u, err := url.Parse(urls)
	if err != nil {
		log.Fatal(err)
	}
	if auth != "" {
		if userpass := strings.Split(auth, ":"); len(userpass) == 2 {
			u.User = url.UserPassword(userpass[0], userpass[1])
		} else {
			u.User = url.User(auth)
		}
	}

	return u
}

func downloadFile(req *Request, res *http.Response, filename string) {
	fd, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0o666)
	if err != nil {
		log.Fatalf("create download file %q failed: %v", filename, err)
	}

	if runtime.GOOS != "windows" {
		fmt.Println(Color(res.Proto, Magenta), Color(res.Status, Green))
		for k, val := range res.Header {
			fmt.Println(Color(k, Gray), ":", Color(strings.Join(val, " "), Cyan))
		}
	} else {
		fmt.Println(res.Proto, res.Status)
		for k, val := range res.Header {
			fmt.Println(k, ":", strings.Join(val, " "))
		}
	}
	fmt.Println("")
	var total int64
	if contentLength := res.Header.Get("Content-Length"); contentLength != "" {
		total, _ = strconv.ParseInt(contentLength, 10, 64)
	}
	fmt.Printf("Downloading to %q\n", filename)
	pb := NewProgressBar(total)
	pb.Start()
	mw := io.MultiWriter(fd, pb)

	if limitRate > 0 {
		res.Body = shapeio.NewReader(res.Body, shapeio.WithRateLimit(float64(limitRate)))
	}

	if _, err := io.Copy(mw, &bodyReader{r: res.Body, conn: req.ConnInfo.Conn}); err != nil {
		log.Fatal("Can't Write the body into file", err)
	}
	pb.Finish()
	iox.Close(fd)
	iox.Close(res.Body)
	fmt.Println()
}

type bodyReader struct {
	r    io.Reader
	conn net.Conn
}

func (b bodyReader) Read(p []byte) (n int, err error) {
	if timeout > 0 {
		t := time.Now().Add(timeout)
		if err := b.conn.SetDeadline(t); err != nil {
			log.Printf("failed to set deadline: %v", err)
		}
	}

	return b.r.Read(p)
}
