package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	"time"

	"github.com/bingoohuang/gg/pkg/fla9"
	"github.com/bingoohuang/gg/pkg/osx"
	"github.com/bingoohuang/gg/pkg/rest"
	"github.com/bingoohuang/gg/pkg/ss"
	"github.com/bingoohuang/gg/pkg/thinktime"
	"github.com/bingoohuang/gg/pkg/v"
	"github.com/bingoohuang/goup"
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

	if len(*Urls) == 0 {
		log.Fatal("Miss the URL")
	}

	stdin := parseStdin()

	start := time.Now()
	for _, urlAddr := range *Urls {
		run(urlAddr, nonFlagArgs, stdin)
	}

	if HasPrintOption(printVerbose) {
		fmt.Println("Complete, total cost: ", time.Since(start))
	}
}

func parseStdin() io.Reader {
	if isWindows() {
		return nil
	}

	stat, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		return os.Stdin
	}

	return nil
}

var uploadFilePb *ProgressBar

func run(urlAddr string, nonFlagArgs []string, stdin io.Reader) {
	u := rest.FixURI(urlAddr,
		rest.WithAuth(auth),
		rest.WithFatalErr(true),
		rest.WithDefaultScheme(ss.If(caFile != "", "https", "http")),
	).Data

	if stdin != nil && *method == http.MethodGet {
		*method = http.MethodPost
	}

	req := getHTTP(*method, u.String(), nonFlagArgs, timeout)
	if u.User != nil {
		password, _ := u.User.Password()
		req.SetBasicAuth(u.User.Username(), password)
	}

	req.Req = req.Req.WithContext(httptrace.WithClientTrace(req.Req.Context(), createClientTrace(req)))

	req.SetTLSClientConfig(createTlsConfig())
	if proxyURL := parseProxyURL(req.Req); proxyURL != nil {
		log.Printf("Proxy URL: %s", proxyURL)
		req.SetProxy(http.ProxyURL(proxyURL))
	}

	if stdin != nil {
		stdinCh := make(chan interface{})
		go readStdin(stdin, stdinCh)
		req.BodyCh(stdinCh)
	}

	thinkerFn := func() {}
	if thinker, _ := thinktime.ParseThinkTime(think); thinker != nil {
		thinkerFn = func() {
			thinker.Think(true)
		}
	}

	req.SetupTransport()
	req.BuildUrl()

	if benchC > 1 { // AB bench
		req.Debug(false)
		RunBench(req, thinkerFn)
		return
	}

	for i := 0; benchN == 0 || i < benchN; i++ {
		start := time.Now()

		err := doRequest(req, u)
		if HasPrintOption(printVerbose) {
			log.Printf("current request cost time: %s", time.Since(start))
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Printf("error: %v", err)
			}
			break
		}
		req.Reset()
		if benchN == 0 || i < benchN-1 {
			thinkerFn()
		}
	}
}

func setBody(req *Request) {
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
}

func readStdin(stdin io.Reader, stdinCh chan interface{}) {
	d := json.NewDecoder(stdin)
	d.UseNumber()

	for {
		var j interface{}
		if err := d.Decode(&j); err != nil {
			if errors.Is(err, io.EOF) {
				close(stdinCh)
			} else {
				log.Println(err)
			}
			return
		}
		stdinCh <- j
	}
}

// Proxy Support
func parseProxyURL(req *http.Request) *url.URL {
	if proxy != "" {
		return rest.FixURI(proxy, rest.WithFatalErr(true)).Data
	}

	p, err := http.ProxyFromEnvironment(req)
	if err != nil {
		log.Fatal("Environment Proxy Url parse err", err)
	}
	return p
}

func createTlsConfig() (tlsConfig *tls.Config) {
	if caFile != "" {
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(osx.ReadFile(caFile, osx.WithFatalOnError(true)).Data)
		tlsConfig = &tls.Config{RootCAs: pool}
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

func doRequest(req *Request, u *url.URL) error {
	if req.bodyCh != nil {
		if err := req.NextBody(); err != nil {
			return err
		}
	} else {
		setBody(req)
	}

	doRequestInternal(req, u)
	return nil
}

func doRequestInternal(req *Request, u *url.URL) {
	if benchN == 0 || benchN > 1 {
		req.Header("Gurl-N", fmt.Sprintf("%d", currentN.Inc()))
	}
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
	download = strings.ToLower(download)

	if download == "no" || download == "n" {
		// do not goto downloading
	} else if (download == "yes" || download == "y") ||
		(cl > 2048 || fn != "" || !ss.ContainsFold(ct, "json", "text", "xml")) {
		if *method != "HEAD" {
			if fn == "" {
				_, fn = path.Split(u.Path)
			}
			if ss.ContainsFold(ct, "json") && !strings.HasSuffix(fn, ".json") {
				fn += ".json"
			} else if ss.ContainsFold(ct, "text") && !strings.HasSuffix(fn, ".txt") {
				fn += ".txt"
			} else if ss.ContainsFold(ct, "xml") && !strings.HasSuffix(fn, ".xml") {
				fn += ".xml"
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
		printResponseForNonWindows(req, res, false)
	}

	if HasPrintOption(printHTTPTrace) {
		req.stat.print(u.Scheme)
	}
}

func printResponseForNonWindows(req *Request, res *http.Response, download bool) {
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
			if !saveTempFile(dumpBody, MaxRequestSize) {
				fmt.Println(formatBytes(dumpBody, pretty, true))
			}
		}
		if HasPrintOption(printRespHeader) {
			fmt.Println(Color(res.Proto, Magenta), Color(res.Status, Green))
			for k, val := range res.Header {
				fmt.Printf("%s: %s\n", Color(k, Gray), Color(strings.Join(val, " "), Cyan))
			}

			if res.Close {
				fmt.Printf("%s: %s\n", Color("Connection", Gray), Color("Close", Cyan))
			}

			fmt.Println()
		}
		if !download && HasPrintOption(printRespBody) {
			fmt.Println(formatResponseBody(req, pretty, true))
		}
	} else if !download {
		b := formatResponseBody(req, pretty, false)
		_, _ = os.Stdout.WriteString(b)
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

func isWindows() bool {
	return runtime.GOOS == "windows"
}
