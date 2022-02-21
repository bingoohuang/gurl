// Gurl is a Go implemented CLI cURL-like tool for humans
// gurl [flags] [METHOD] URL [ITEM [ITEM]]
package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/bingoohuang/gg/pkg/ss"
	"github.com/bingoohuang/gg/pkg/v"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/bingoohuang/gg/pkg/fla9"
)

const (
	printReqHeader uint8 = 1 << iota
	printReqBody
	printRespHeader
	printRespBody
)

var (
	ver, form, pretty, raw, download, insecureSSL bool
	auth, proxy, printV, body                     string
	printOption                                   uint8
	benchN, benchC                                int
	timeout                                       time.Duration

	isjson  = fla9.Bool("json,j", true, "Send the data as a JSON object")
	method  = fla9.String("method,m", "GET", "HTTP method")
	Urls    = flagEnv("url,u", "", "HTTP request URL")
	jsonmap = map[string]interface{}{}
)

func init() {
	fla9.BoolVar(&ver, "v", false, "Print Version Number")
	fla9.BoolVar(&raw, "raw,r", false, "Print JSON Raw Format")
	fla9.StringVar(&printV, "print,p", "A", "Print request and response")
	fla9.BoolVar(&form, "f", false, "Submitting as a form")
	fla9.BoolVar(&download, "d", false, "Download the url content as file")
	fla9.BoolVar(&insecureSSL, "i", false, "Allow connections to SSL sites without certs")
	fla9.DurationVar(&timeout, "t", 1*time.Minute, "Timeout for read and write")

	flagEnvVar(&auth, "auth", "", "HTTP authentication username:password, USER[:PASS]")
	flagEnvVar(&proxy, "proxy", "", "Proxy host and port, PROXY_URL")
	fla9.IntVar(&benchN, "b.n", 0, "Number of bench requests to run")
	fla9.IntVar(&benchC, "b.c", 100, "Number of bench requests to run concurrently.")
	fla9.StringVar(&body, "body,b", "", "Raw data send as body")
}

func parsePrintOption(s string) {
	AdjustPrintOption(s, 'A', printReqHeader|printReqBody|printRespHeader|printRespBody)
	AdjustPrintOption(s, 'a', printReqHeader|printReqBody|printRespHeader|printRespBody)
	AdjustPrintOption(s, 'H', printReqHeader)
	AdjustPrintOption(s, 'B', printReqBody)
	AdjustPrintOption(s, 'h', printRespHeader)
	AdjustPrintOption(s, 'b', printRespBody)
}

func AdjustPrintOption(s string, r rune, flags uint8) {
	if strings.ContainsRune(s, r) {
		printOption |= flags
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
	if ver {
		fmt.Println(v.Version())
		os.Exit(2)
	}

	pretty = !raw
	nonFlagArgs := filter(fla9.Args())

	parsePrintOption(printV)
	if !HasPrintOption(printReqBody) {
		defaultSetting.DumpBody = false
	}

	stdin := parseStdin()

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

func run(urlAddr string, nonFlagArgs []string, stdin []byte) {
	u := parseURL(urlAddr)
	urlAddr = u.String()
	req := getHTTP(*method, urlAddr, nonFlagArgs, timeout)
	if u.User != nil {
		password, _ := u.User.Password()
		req.SetBasicAuth(u.User.Username(), password)
	}
	// Insecure SSL Support
	if insecureSSL {
		req.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}
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
	if body != "" {
		req.Body(body)
	}
	if len(stdin) > 0 {
		var j interface{}
		d := json.NewDecoder(bytes.NewReader(stdin))
		d.UseNumber()
		if err := d.Decode(&j); err != nil {
			req.Body(stdin)
		} else {
			req.JsonBody(j)
		}
	}

	// AB bench
	if benchN > 0 {
		req.Debug(false)
		RunBench(req)
		return
	}
	res, err := req.Response()
	if err != nil {
		log.Fatalln("can't get the url", err)
	}

	fn := ""
	if d := res.Header.Get("Content-Disposition"); d != "" {
		if _, params, _ := mime.ParseMediaType(d); params != nil {
			fn = params["filename"]
		}
	}
	ct := res.Header.Get("Content-Type")
	if download || fn != "" || !ss.ContainsFold(ct, "json", "text", "xml") {
		downloadFile(u, res, fn)
		return
	}

	if runtime.GOOS != "windows" {
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

			if HasPrintOption(printReqHeader) {
				fmt.Println(ColorfulRequest(string(dumpHeader)))
			}
			if HasPrintOption(printReqBody) {
				fmt.Println(string(dumpBody))
			}

			if HasPrintOption(printRespHeader) {
				fmt.Println(Color(res.Proto, Magenta), Color(res.Status, Green))
				for k, v := range res.Header {
					fmt.Printf("%s: %s\n", Color(k, Gray), Color(strings.Join(v, " "), Cyan))
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
	} else {
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
			for k, v := range res.Header {
				fmt.Println(k, ":", strings.Join(v, " "))
			}
			fmt.Println("")
		}
		if HasPrintOption(printRespBody) {
			fmt.Println(formatResponseBody(req, pretty, false))
		}
	}
}

func parseURL(urls string) *url.URL {
	if urls == "" {
		usage()
	}
	if strings.HasPrefix(urls, ":") {
		urlb := []byte(urls)
		if urls == ":" {
			urls = "http://localhost/"
		} else if len(urls) > 1 && urlb[1] != '/' {
			urls = "http://localhost" + urls
		} else {
			urls = "http://localhost" + string(urlb[1:])
		}
	}
	if !strings.HasPrefix(urls, "http://") && !strings.HasPrefix(urls, "https://") {
		urls = "http://" + urls
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

func downloadFile(u *url.URL, res *http.Response, filename string) {
	if filename == "" {
		_, filename = filepath.Split(u.Path)
	}
	fd, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0o666)
	if err != nil {
		log.Fatal("can't create file", err)
	}
	if runtime.GOOS != "windows" {
		fmt.Println(Color(res.Proto, Magenta), Color(res.Status, Green))
		for k, v := range res.Header {
			fmt.Println(Color(k, Gray), ":", Color(strings.Join(v, " "), Cyan))
		}
	} else {
		fmt.Println(res.Proto, res.Status)
		for k, v := range res.Header {
			fmt.Println(k, ":", strings.Join(v, " "))
		}
	}
	fmt.Println("")
	var total int64
	if contentLength := res.Header.Get("Content-Length"); contentLength != "" {
		total, _ = strconv.ParseInt(contentLength, 10, 64)
	}
	fmt.Printf("Downloading to \"%s\"\n", filename)
	pb := NewProgressBar(total)
	pb.Start()
	mw := io.MultiWriter(fd, pb)
	if _, err := io.Copy(mw, res.Body); err != nil {
		log.Fatal("Can't Write the body into file", err)
	}
	pb.Finish()
	fd.Close()
	res.Body.Close()
}