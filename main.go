package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/bingoohuang/gg/pkg/fla9"
	"github.com/bingoohuang/goup"
	"github.com/bingoohuang/goup/shapeio"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"

	"github.com/bingoohuang/gg/pkg/ss"
	"github.com/bingoohuang/gg/pkg/thinktime"
	"github.com/bingoohuang/gg/pkg/v"
)

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
	if isWindows() {
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

	req.SetTLSClientConfig(createTlsConfig())
	req.SetProxy(http.ProxyURL(parseProxyURL(req.Req)))

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

	thinkerFn := func() {}
	if thinker, _ := thinktime.ParseThinkTime(think); thinker != nil {
		thinkerFn = func() {
			thinker.Think(true)
		}
	}

	req.SetupTransport()

	if benchC > 1 { // AB bench
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

// Proxy Support
func parseProxyURL(req *http.Request) *url.URL {
	if proxy != "" {
		proxyURI, err := FixURI(proxy, "")
		if err != nil {
			log.Fatalf("Fix Proxy Url failed: %v", err)
		}
		purl, err := url.Parse(proxyURI)
		if err != nil {
			log.Fatalf("Proxy Url parse failed: %v", err)
		}
		return purl
	}

	eurl, err := http.ProxyFromEnvironment(req)
	if err != nil {
		log.Fatal("Environment Proxy Url parse err", err)
	}
	return eurl
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
	cl, _ := strconv.ParseInt(res.Header.Get("Content-Length"), 10, 64)
	ct := res.Header.Get("Content-Type")
	if download || cl > 2048 || fn != "" || !ss.ContainsFold(ct, "json", "text", "xml") {
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

	if isWindows() {
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
		dps := strings.Split(string(req.Dump), "\n")
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
	dps := strings.Split(string(req.Dump), "\n")
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
		if urls == ":" {
			urls = schema + "://localhost/"
		} else if len(urls) > 1 && urls[1] != '/' {
			urls = schema + "://localhost" + urls
		} else {
			urls = schema + "://localhost" + urls[1:]
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

func isWindows() bool {
	return runtime.GOOS == "windows"
}
