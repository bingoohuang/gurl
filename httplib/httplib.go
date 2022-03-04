// Copyright 2014 beego Author. All Rights Reserved.
// Copyright 2015 bat authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Usage:
//
// import "github.com/astaxie/beego/httplib"
//
//	b := httplib.Post("http://beego.me/")
//	b.Param("username","astaxie")
//	b.Param("password","123456")
//	b.PostFile("uploadfile1", "httplib.pdf")
//	b.PostFile("uploadfile2", "httplib.txt")
//	str, err := b.String()
//	if err != nil {
//		t.Fatal(err)
//	}
//	fmt.Println(str)
//
//  more docs http://beego.me/docs/module/httplib.md
package httplib

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptrace"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

var defaultSetting = Settings{
	UserAgent:      "beegoServer",
	ConnectTimeout: 60 * time.Second,
	Gzip:           true,
	DumpBody:       true,
}

var (
	defaultCookieJar http.CookieJar
	settingMutex     sync.Mutex
)

// createDefaultCookie creates a global cookiejar to store cookies.
func createDefaultCookie() {
	settingMutex.Lock()
	defer settingMutex.Unlock()
	defaultCookieJar, _ = cookiejar.New(nil)
}

// SetDefaultSetting Overwrite default settings
func SetDefaultSetting(setting Settings) {
	settingMutex.Lock()
	defer settingMutex.Unlock()
	defaultSetting = setting
}

// NewRequest return *Request with specific method
func NewRequest(rawurl, method string) *Request {
	var resp http.Response
	u, err := url.Parse(rawurl)
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
		url:     rawurl,
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

	if trans == nil {
		// create default transport
		trans = &http.Transport{
			TLSClientConfig: b.Setting.TlsClientConfig,
			Proxy:           b.Setting.Proxy,
			DialContext:     TimeoutDialer(b.Setting.ConnectTimeout),
		}
	} else {
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
	}

	// https://blog.witd.in/2019/02/25/golang-http-client-关闭重用连接两种方法/
	if t, ok := trans.(*http.Transport); ok {
		t.DisableKeepAlives = b.DisableKeepAlives
	}
	b.Req.Close = b.DisableKeepAlives
	b.Transport = trans
}

// Get returns *Request with GET method.
func Get(url string) *Request { return NewRequest(url, "GET") }

// Post returns *Request with POST method.
func Post(url string) *Request { return NewRequest(url, "POST") }

// Put returns *Request with PUT method.
func Put(url string) *Request { return NewRequest(url, "PUT") }

// Delete returns *Request DELETE method.
func Delete(url string) *Request { return NewRequest(url, "DELETE") }

// Head returns *Request with HEAD method.
func Head(url string) *Request { return NewRequest(url, "HEAD") }

// Settings .
type Settings struct {
	ShowDebug       bool
	UserAgent       string
	ConnectTimeout  time.Duration
	TlsClientConfig *tls.Config
	Proxy           func(*http.Request) (*url.URL, error)
	Transport       http.RoundTripper
	EnableCookie    bool
	Gzip            bool
	DumpBody        bool
}

// Request provides more useful methods for requesting one url than http.Request.
type Request struct {
	url     string
	Req     *http.Request
	queries map[string]string
	params  map[string]string
	files   map[string]string
	Setting Settings
	resp    *http.Response
	body    []byte
	dump    []byte

	DisableKeepAlives bool
	Transport         http.RoundTripper

	ConnInfo httptrace.GotConnInfo
	Rewinder func()
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

// DumpRequest return the DumpRequest
func (b *Request) DumpRequest() []byte {
	return b.dump
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
// 		u, _ := url.ParseRequestURI("http://127.0.0.1:8118")
// 		return u, nil
// 	}
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

func (b *Request) BodyAndSize(body io.Reader, size int64) *Request {
	b.Req.Body = ioutil.NopCloser(body)
	b.Req.ContentLength = size

	return b
}

// Body adds request raw body.
// it supports string and []byte.
func (b *Request) Body(data interface{}) *Request {
	switch t := data.(type) {
	case string:
		bf := bytes.NewBufferString(t)
		b.BodyAndSize(bf, int64(len(t)))
	case []byte:
		bf := bytes.NewBuffer(t)
		b.BodyAndSize(bf, int64(len(t)))
	}
	return b
}

// JsonBody adds request raw body encoding by JSON.
func (b *Request) JsonBody(obj interface{}) (*Request, error) {
	if b.Req.Body == nil && obj != nil {
		buf := bytes.NewBuffer(nil)
		enc := json.NewEncoder(buf)
		if err := enc.Encode(obj); err != nil {
			return b, err
		}
		b.Req.Body = ioutil.NopCloser(buf)
		b.Req.ContentLength = int64(buf.Len())
		b.Req.Header.Set("Content-Type", "application/json")
	}
	return b, nil
}

func appendUrl(url, append string) string {
	if strings.Index(url, "?") != -1 {
		return url + "&" + append
	}

	return url + "?" + append
}

func (b *Request) buildUrl() {
	paramBody := createParamBody(b.params)
	queryBody := createParamBody(b.queries)
	b.url = appendUrl(b.url, queryBody)

	// build GET url with query string
	if b.Req.Method == "GET" && len(paramBody) > 0 {
		b.url = appendUrl(b.url, paramBody)
		return
	}

	// build POST/PUT/PATCH url and body
	if (b.Req.Method == "POST" || b.Req.Method == "PUT" || b.Req.Method == "PATCH") && b.Req.Body == nil {
		// with files
		if len(b.files) > 0 {
			pr, pw := io.Pipe()
			bodyWriter := multipart.NewWriter(pw)
			go func() {
				for formname, filename := range b.files {
					fileWriter, err := bodyWriter.CreateFormFile(formname, filename)
					if err != nil {
						log.Fatal(err)
					}
					fh, err := os.Open(filename)
					if err != nil {
						log.Fatal(err)
					}
					// iocopy
					_, err = io.Copy(fileWriter, fh)
					fh.Close()
					if err != nil {
						log.Fatal(err)
					}
				}
				for k, v := range b.params {
					bodyWriter.WriteField(k, v)
				}
				bodyWriter.Close()
				pw.Close()
			}()
			contentType := bodyWriter.FormDataContentType()
			b.Setting.DumpBody = false
			b.Header("Content-Type", contentType)
			b.Req.Body = ioutil.NopCloser(pr)
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
	b.body = nil
	if b.Rewinder != nil {
		b.Rewinder()
	}
}

func (b *Request) Response() (*http.Response, error) {
	if b.resp.StatusCode != 0 {
		return b.resp, nil
	}
	resp, err := b.SendOut()
	if err != nil {
		return nil, err
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

func (b *Request) SendOut() (*http.Response, error) {
	b.buildUrl()
	url, err := url.Parse(b.url)
	if err != nil {
		return nil, err
	}

	b.Req.URL = url

	var jar http.CookieJar = nil
	if b.Setting.EnableCookie {
		if defaultCookieJar == nil {
			createDefaultCookie()
		}
		jar = defaultCookieJar
	}

	client := &http.Client{
		Transport: b.Transport,
		Jar:       jar,
	}

	if b.Setting.UserAgent != "" && b.Req.Header.Get("User-Agent") == "" {
		b.Req.Header.Set("User-Agent", b.Setting.UserAgent)
	}

	if b.Setting.ShowDebug {
		dump, err := httputil.DumpRequest(b.Req, b.Setting.DumpBody)
		if err != nil {
			println(err.Error())
		}
		b.dump = dump
	}

	return client.Do(b.Req)
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
	if b.body != nil {
		return b.body, nil
	}
	resp, err := b.Response()
	if err != nil {
		return nil, err
	}
	if resp.Body == nil {
		return nil, nil
	}
	defer resp.Body.Close()
	if b.Setting.Gzip && resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		b.body, err = ioutil.ReadAll(reader)
	} else {
		b.body, err = ioutil.ReadAll(resp.Body)
	}
	if err != nil {
		return nil, err
	}
	return b.body, nil
}

// ToFile saves the body data in response to one file.
// it calls Response inner.
func (b *Request) ToFile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	resp, err := b.Response()
	if err != nil {
		return err
	}
	if resp.Body == nil {
		return nil
	}
	defer resp.Body.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// ToJson returns the map that marshals from the body bytes as json in response .
// it calls Response inner.
func (b *Request) ToJson(v interface{}) error {
	data, err := b.Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// ToXml returns the map that marshals from the body bytes as xml in response .
// it calls Response inner.
func (b *Request) ToXml(v interface{}) error {
	data, err := b.Bytes()
	if err != nil {
		return err
	}
	return xml.Unmarshal(data, v)
}

// TimeoutDialer returns functions of connection dialer with timeout settings for http.Transport Dial field.
func TimeoutDialer(cTimeout time.Duration) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return net.DialTimeout(network, addr, cTimeout)
	}
}
