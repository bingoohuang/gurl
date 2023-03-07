package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"io"
	"log"
	"math/rand"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptrace"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/bingoohuang/gg/pkg/filex"
	"github.com/bingoohuang/gg/pkg/iox"
	"github.com/bingoohuang/goup/shapeio"
	"github.com/bingoohuang/jj"
)

// NewRequest return *Request with specific method
func NewRequest(rawURL, method string) *Request {
	var resp http.Response
	u, err := url.Parse(rawURL)
	if err != nil {
		log.Fatal(err)
	}
	req := http.Request{
		URL:        u,
		Method:     method,
		Header:     make(http.Header),
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}
	return &Request{
		url:     rawURL,
		Req:     &req,
		queries: map[string]string{},
		params:  map[string]string{},
		files:   map[string]string{},
		Setting: defaultSetting,
		resp:    &resp,
	}
}

func (b *Request) SetupTransport() {
	trans := b.Setting.Transport
	if trans == nil { // create default transport
		trans = &http.Transport{}
	}

	// if b.transport is *http.Transport then set the settings.
	if t, ok := trans.(*http.Transport); ok {
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = b.Setting.TlsClientConfig
		}
		if t.Proxy == nil {
			t.Proxy = b.Setting.Proxy
		}
		if t.DialContext == nil {
			t.DialContext = TimeoutDialer(b.Setting.ConnectTimeout)
		}
	}

	// https://blog.witd.in/2019/02/25/golang-http-client-关闭重用连接两种方法/
	if t, ok := trans.(*http.Transport); ok {
		t.DisableKeepAlives = b.DisableKeepAlives
	}
	b.Req.Close = b.DisableKeepAlives
	b.Transport = trans
}

// Settings .
type Settings struct {
	ShowDebug       bool
	UserAgent       string
	ConnectTimeout  time.Duration
	TlsClientConfig *tls.Config
	Proxy           func(*http.Request) (*url.URL, error)
	Transport       http.RoundTripper
	EnableCookie    bool
	DumpBody        bool
}

// Request provides more useful methods for requesting one url than http.Request.
type Request struct {
	url      string
	urlQuery []string

	Req  *http.Request
	resp *http.Response

	queries, params, files map[string]string

	Setting          Settings
	rspBody, reqDump []byte

	DisableKeepAlives bool

	Transport http.RoundTripper
	ConnInfo  httptrace.GotConnInfo

	bodyCh        chan string
	stat          *httpStat
	DryRequest    bool
	Timeout       time.Duration
	cancelTimeout context.CancelFunc
	timeResetCh   chan struct{}
}

// SetBasicAuth sets the request's Authorization header to use HTTP Basic Authentication with the provided username and password.
func (b *Request) SetBasicAuth(username, password string) *Request {
	b.Req.SetBasicAuth(username, password)
	return b
}

// SetEnableCookie sets enable/disable cookiejar
func (b *Request) SetEnableCookie(enable bool) *Request {
	b.Setting.EnableCookie = enable
	return b
}

// SetUserAgent sets User-Agent header field
func (b *Request) SetUserAgent(useragent string) *Request {
	b.Setting.UserAgent = useragent
	return b
}

// Debug sets show debug or not when executing request.
func (b *Request) Debug(isdebug bool) *Request {
	b.Setting.ShowDebug = isdebug
	return b
}

// DumpBody Dump Body.
func (b *Request) DumpBody(isdump bool) *Request {
	b.Setting.DumpBody = isdump
	return b
}

// SetTimeout sets connect time out and read-write time out for BeegoRequest.
func (b *Request) SetTimeout(connectTimeout time.Duration) *Request {
	b.Setting.ConnectTimeout = connectTimeout
	return b
}

// SetTLSClientConfig sets tls connection configurations if visiting https url.
func (b *Request) SetTLSClientConfig(config *tls.Config) *Request {
	b.Setting.TlsClientConfig = config
	return b
}

// Header add header item string in request.
func (b *Request) Header(key, value string) *Request {
	b.Req.Header.Set(key, value)
	return b
}

// SetHost Set HOST
func (b *Request) SetHost(host string) *Request {
	b.Req.Host = host
	return b
}

// SetProtocolVersion Set the protocol version for incoming requests.
// Client requests always use HTTP/1.1.
func (b *Request) SetProtocolVersion(vers string) *Request {
	if len(vers) == 0 {
		vers = "HTTP/1.1"
	}

	major, minor, ok := http.ParseHTTPVersion(vers)
	if ok {
		b.Req.Proto = vers
		b.Req.ProtoMajor = major
		b.Req.ProtoMinor = minor
	}

	return b
}

// SetCookie add cookie into request.
func (b *Request) SetCookie(cookie *http.Cookie) *Request {
	b.Req.Header.Add("Cookie", cookie.String())
	return b
}

// SetTransport Set transport to
func (b *Request) SetTransport(transport http.RoundTripper) *Request {
	b.Setting.Transport = transport
	return b
}

// SetProxy Set http proxy
// example:
//
//	func(Req *http.Request) (*url.URL, error) {
//		u, _ := url.ParseRequestURI("http://127.0.0.1:8118")
//		return u, nil
//	}
func (b *Request) SetProxy(proxy func(*http.Request) (*url.URL, error)) *Request {
	b.Setting.Proxy = proxy
	return b
}

// Param adds query param in to request.
// params build query string as ?key1=value1&key2=value2...
func (b *Request) Param(key, value string) *Request {
	b.params[key] = value
	return b
}

// Query adds query param in to request.
// params build query string as ?key1=value1&key2=value2...
func (b *Request) Query(key, value string) *Request {
	b.queries[key] = value
	return b
}

func (b *Request) PostFile(formname, filename string) *Request {
	b.files[formname] = filename
	return b
}

func (b *Request) BodyAndSize(body io.ReadCloser, size int64) *Request {
	b.Req.Body = body
	b.Req.ContentLength = size

	return b
}

// BodyCh set body channel..
func (b *Request) BodyCh(data chan string) *Request {
	b.bodyCh = data
	return b
}

func (b *Request) evalBytes(data []byte) (io.ReadCloser, int64) {
	eval := Eval(string(data))
	if jj.Valid(eval) {
		b.Header("Content-Type", "application/json")
	}
	return io.NopCloser(bytes.NewBufferString(eval)), int64(len(eval))
}

func (b *Request) BodyFileLines(t string) bool {
	if strings.HasPrefix(t, "@") {
		t = t[1:]
	}

	if lineMode := strings.HasSuffix(t, ":line"); lineMode {
		if fn := strings.TrimSuffix(t, ":line"); filex.Exists(fn) {
			t = strings.TrimSuffix(t, ":line")
			lines, err := filex.LinesChan(t, 1000)
			if err != nil {
				log.Fatalf("E! create line chan for %s, failed: %v", t, err)
			}
			b.BodyCh(lines)
			return true
		}
	}

	return false
}

func (b *Request) Body(data interface{}) *Request {
	switch t := data.(type) {
	case string:
		filename := t
		if at := strings.HasPrefix(t, "@"); at {
			filename = t[1:]
		}
		if stat, _ := os.Stat(filename); stat != nil && !stat.IsDir() {
			file, err := os.Open(filename)
			if err != nil {
				log.Fatalf("open %s failed: %v", filename, err)
			}
			b.BodyAndSize(file, stat.Size())
			return b
		}

		b.BodyAndSize(b.evalBytes([]byte(t)))
	case []byte:
		b.BodyAndSize(b.evalBytes(t))
	}
	return b
}

func (b *Request) NextBody() (err error) {
	if b.bodyCh != nil {
		d, ok := <-b.bodyCh
		if !ok {
			b.bodyCh = nil
			return io.EOF
		}
		b.BodyString(d)
		return
	}

	return io.EOF
}

// JsonBody adds request raw body encoding by JSON.
func (b *Request) JsonBody(obj interface{}) (*Request, error) {
	if obj != nil {
		buf := bytes.NewBuffer(nil)
		enc := json.NewEncoder(buf)
		if err := enc.Encode(obj); err != nil {
			return b, err
		}

		b.BodyString(buf.String())
	}
	return b, nil
}

func (b *Request) BodyString(s string) {
	eval := Eval(s)
	b.Req.Body = io.NopCloser(strings.NewReader(eval))
	b.Req.ContentLength = int64(len(eval))
	if jj.Valid(s) {
		b.Header("Content-Type", "application/json")
	}
}

func appendUrl(url, append string) string {
	if append == "" {
		return url
	}

	if strings.Contains(url, "?") {
		return url + "&" + append
	}

	return url + "?" + append
}

func (b *Request) BuildUrl() {
	if queryBody := createParamBody(b.queries); queryBody != "" {
		b.urlQuery = append(b.urlQuery, queryBody)
	}

	paramBody := createParamBody(b.params)
	// build GET url with query string
	if b.Req.Method == "GET" && len(paramBody) > 0 {
		b.urlQuery = append(b.urlQuery, paramBody)
		return
	}

	// build POST/PUT/PATCH url and body
	if (b.Req.Method == "POST" || b.Req.Method == "PUT" || b.Req.Method == "PATCH") && b.Req.Body == nil {
		// with files
		if len(b.files) > 0 {
			pr, pw := io.Pipe()
			bodyWriter := multipart.NewWriter(pw)
			go func() {
				for formName, filename := range b.files {
					fileWriter, err := bodyWriter.CreateFormFile(formName, filename)
					if err != nil {
						log.Fatal(err)
					}
					fh, err := os.Open(filename)
					if err != nil {
						log.Fatal(err)
					}
					_, err = io.Copy(fileWriter, fh)
					iox.Close(fh)
					if err != nil {
						log.Fatal(err)
					}
				}
				for k, v := range b.params {
					_ = bodyWriter.WriteField(k, v)
				}
				iox.Close(bodyWriter)
				iox.Close(pw)
			}()
			contentType := bodyWriter.FormDataContentType()
			b.Setting.DumpBody = false
			b.Header("Content-Type", contentType)
			b.Req.Body = io.NopCloser(pr)
			return
		}

		// with params
		if len(paramBody) > 0 {
			b.Header("Content-Type", "application/x-www-form-urlencoded")
			b.Body(paramBody)
		}
	}
}

func (b *Request) Reset() {
	b.resp.StatusCode = 0
	b.rspBody = nil
	if b.timeResetCh != nil {
		select {
		case b.timeResetCh <- struct{}{}:
		default:
		}
	}
	valuer.ClearCache()
}

func (b *Request) Response() (*http.Response, error) {
	if b.resp.StatusCode != 0 {
		return b.resp, nil
	}

	resp, err := b.SendOut()
	if err != nil {
		return nil, err
	}

	if limitRate.IsForRsp() && resp.Body != nil {
		resp.Body = shapeio.NewReader(resp.Body, shapeio.WithRateLimit(limitRate.Float64()))
	}

	b.resp = resp

	return resp, nil
}

func createParamBody(params map[string]string) string {
	var paramBody string
	if len(params) > 0 {
		var buf bytes.Buffer
		for k, v := range params {
			buf.WriteString(url.QueryEscape(k))
			buf.WriteByte('=')
			buf.WriteString(url.QueryEscape(v))
			buf.WriteByte('&')
		}
		paramBody = buf.String()
		paramBody = paramBody[0 : len(paramBody)-1]
	}

	return paramBody
}

// LogRedirects log redirect
// refer: Go HTTP Redirect的知识点总结 https://colobu.com/2017/04/19/go-http-redirect/
type LogRedirects struct {
	http.RoundTripper
}

func (l LogRedirects) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	t := l.RoundTripper
	if t == nil {
		t = http.DefaultTransport
	}
	resp, err = t.RoundTrip(req)
	if err != nil {
		return
	}
	if isRedirect(resp.StatusCode) && HasPrintOption(printVerbose) {
		log.Printf("FROM %s", req.URL)
		log.Printf("Redirect(%d) to %s", resp.StatusCode, resp.Header.Get("Location"))
	}

	return
}

func (b *Request) SendOut() (*http.Response, error) {
	full := b.url
	for _, q := range b.urlQuery {
		full = appendUrl(full, q)
	}

	u, err := url.Parse(full)
	if err != nil {
		return nil, err
	}

	b.Req.URL = u

	var jar http.CookieJar
	if b.Setting.EnableCookie {
		jar, _ = cookiejar.New(nil)
	}

	client := &http.Client{
		Transport: LogRedirects{RoundTripper: b.Transport},
		Jar:       jar,
	}

	if b.Setting.UserAgent != "" && b.Req.Header.Get("User-Agent") == "" {
		b.Header("User-Agent", b.Setting.UserAgent)
	}

	if b.Req.Body != nil && gzipOn {
		b.Req.ContentLength = -1
		b.Req.Header.Del("Content-Length")
		b.Header("Transfer-Encoding", "chunked")
		b.Header("Content-Encoding", "gzip")
	}

	// -proxy http://username:password@127.0.0.1:7777 可以干活，不需要下面的独立设置
	/*
		if b.Setting.Proxy != nil { // adding proxy authentication
			basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(("username:password")))
			b.Req.Header.Add("Proxy-Authorization", basicAuth)
		}
	*/

	if b.Setting.ShowDebug {
		dump, err := httputil.DumpRequest(b.Req, b.Setting.DumpBody)
		if err != nil {
			println(err.Error())
		}
		b.reqDump = dump
	}

	if b.Req.Body != nil && gzipOn {
		b.Req.Body = NewGzipReader(b.Req.Body)
	}

	if limitRate.IsForReq() && b.Req.Body != nil {
		b.Req.Body = shapeio.NewReader(b.Req.Body, shapeio.WithRateLimit(limitRate.Float64()))
	}

	if b.DryRequest {
		return &http.Response{}, nil
	}

	req := b.Req

	return client.Do(req)
}

func NewGzipReader(source io.Reader) *io.PipeReader {
	r, w := io.Pipe()
	go func() {
		defer w.Close()

		zip := gzip.NewWriter(w)
		defer zip.Close()

		io.Copy(zip, source)
	}()
	return r
}

// String returns the body string in response.
// it calls Response inner.
func (b *Request) String() (string, error) {
	data, err := b.Bytes()
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// Bytes returns the body []byte in response.
// it calls Response inner.
func (b *Request) Bytes() ([]byte, error) {
	if b.rspBody != nil {
		return b.rspBody, nil
	}
	resp, err := b.Response()
	if err != nil {
		return nil, err
	}
	if resp.Body == nil {
		return nil, nil
	}
	defer iox.Close(resp.Body)
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err1 := gzip.NewReader(resp.Body)
		if err1 != nil {
			return nil, err1
		}
		b.rspBody, err = io.ReadAll(reader)
	} else {
		b.rspBody, err = io.ReadAll(resp.Body)
	}
	if err != nil {
		return nil, err
	}
	return b.rspBody, nil
}

// ToFile saves the body data in response to one file.
// it calls Response inner.
func (b *Request) ToFile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer iox.Close(f)

	resp, err := b.Response()
	if err != nil {
		return err
	}
	if resp.Body == nil {
		return nil
	}
	defer iox.Close(resp.Body)
	_, err = io.Copy(f, resp.Body)
	return err
}

// ToJSON returns the map that marshals from the body bytes as json in response .
// it calls Response inner.
func (b *Request) ToJSON(v interface{}) error {
	data, err := b.Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// ToXML returns the map that marshals from the body bytes as xml in response .
// it calls Response inner.
func (b *Request) ToXML(v interface{}) error {
	data, err := b.Bytes()
	if err != nil {
		return err
	}
	return xml.Unmarshal(data, v)
}

// TimeoutDialer returns functions of connection dialer with timeout settings for http.Transport Dial field.
func TimeoutDialer(cTimeout time.Duration) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialer := &net.Dialer{Timeout: cTimeout}
		dnsIP, dnsPort, err := net.SplitHostPort(dns)
		if err != nil {
			dnsIP = dns
			dnsPort = "53"
		}

		if dns != "" {
			addrHost, addrPort, err := net.SplitHostPort(addr)
			if err != nil {
				addrHost = addr
				addrPort = "80"
			}
			if net.ParseIP(addrHost) == nil { // not an IP
				dnsServer := net.JoinHostPort(dnsIP, dnsPort)
				ips, err := Resolve(addrHost, dnsServer)
				if err != nil {
					log.Fatalf("resolve %s by dns server: %s failed: %v", addrHost, dnsServer, err)
				}
				if len(ips) > 0 {
					rand.Seed(time.Now().UnixNano())
					rand.Shuffle(len(ips), func(i, j int) { ips[i], ips[j] = ips[j], ips[i] })
					addr = net.JoinHostPort(ips[0], addrPort)
				}
			}
		}

		return dialer.DialContext(ctx, network, addr)
	}
}

// Resolve resolves host www.google.co by dnsServer like 8.8.8.8:5
func Resolve(host, dnsServer string) ([]string, error) {
	// https://stackoverflow.com/questions/59889882/specifying-dns-server-for-lookup-in-go
	// more https://github.com/Focinfi/go-dns-resolver
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Millisecond * time.Duration(10000),
			}
			return d.DialContext(ctx, network, dnsServer)
		},
	}
	addrs, err := r.LookupIP(context.Background(), "ip4", host)
	ipv4s := make([]string, len(addrs))
	for i, addr := range addrs {
		ipv4s[i] = addr.String()
	}

	return ipv4s, err
}
