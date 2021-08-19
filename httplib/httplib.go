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
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

var defaultSetting = BeegoHttpSettings{false, "beegoServer", 60 * time.Second, 60 * time.Second, nil, nil, nil, false, true, true}
var defaultCookieJar http.CookieJar
var settingMutex sync.Mutex

// createDefaultCookie creates a global cookiejar to store cookies.
func createDefaultCookie() {
	settingMutex.Lock()
	defer settingMutex.Unlock()
	defaultCookieJar, _ = cookiejar.New(nil)
}

// SetDefaultSetting Overwrite default settings
func SetDefaultSetting(setting BeegoHttpSettings) {
	settingMutex.Lock()
	defer settingMutex.Unlock()
	defaultSetting = setting
	if defaultSetting.ConnectTimeout == 0 {
		defaultSetting.ConnectTimeout = 60 * time.Second
	}
	if defaultSetting.ReadWriteTimeout == 0 {
		defaultSetting.ReadWriteTimeout = 60 * time.Second
	}
}

// NewRequest return *BeegoHttpRequest with specific method
func NewRequest(rawurl, method string) *BeegoHttpRequest {
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
	return &BeegoHttpRequest{
		url:     rawurl,
		Req:     &req,
		params:  map[string]string{},
		files:   map[string]string{},
		Setting: defaultSetting,
		resp:    &resp,
	}
}

// Get returns *BeegoHttpRequest with GET method.
func Get(url string) *BeegoHttpRequest {
	return NewRequest(url, "GET")
}

// Post returns *BeegoHttpRequest with POST method.
func Post(url string) *BeegoHttpRequest {
	return NewRequest(url, "POST")
}

// Put returns *BeegoHttpRequest with PUT method.
func Put(url string) *BeegoHttpRequest {
	return NewRequest(url, "PUT")
}

// Delete returns *BeegoHttpRequest DELETE method.
func Delete(url string) *BeegoHttpRequest {
	return NewRequest(url, "DELETE")
}

// Head returns *BeegoHttpRequest with HEAD method.
func Head(url string) *BeegoHttpRequest {
	return NewRequest(url, "HEAD")
}

// BeegoHttpSettings .
type BeegoHttpSettings struct {
	ShowDebug        bool
	UserAgent        string
	ConnectTimeout   time.Duration
	ReadWriteTimeout time.Duration
	TlsClientConfig  *tls.Config
	Proxy            func(*http.Request) (*url.URL, error)
	Transport        http.RoundTripper
	EnableCookie     bool
	Gzip             bool
	DumpBody         bool
}

// BeegoHttpRequest provides more useful methods for requesting one url than http.Request.
type BeegoHttpRequest struct {
	url     string
	Req     *http.Request
	params  map[string]string
	files   map[string]string
	Setting BeegoHttpSettings
	resp    *http.Response
	body    []byte
	dump    []byte
}

// SetBasicAuth sets the request's Authorization header to use HTTP Basic Authentication with the provided username and password.
func (b *BeegoHttpRequest) SetBasicAuth(username, password string) *BeegoHttpRequest {
	b.Req.SetBasicAuth(username, password)
	return b
}

// SetEnableCookie sets enable/disable cookiejar
func (b *BeegoHttpRequest) SetEnableCookie(enable bool) *BeegoHttpRequest {
	b.Setting.EnableCookie = enable
	return b
}

// SetUserAgent sets User-Agent header field
func (b *BeegoHttpRequest) SetUserAgent(useragent string) *BeegoHttpRequest {
	b.Setting.UserAgent = useragent
	return b
}

// Debug sets show debug or not when executing request.
func (b *BeegoHttpRequest) Debug(isdebug bool) *BeegoHttpRequest {
	b.Setting.ShowDebug = isdebug
	return b
}

// DumpBody Dump Body.
func (b *BeegoHttpRequest) DumpBody(isdump bool) *BeegoHttpRequest {
	b.Setting.DumpBody = isdump
	return b
}

// DumpRequest return the DumpRequest
func (b *BeegoHttpRequest) DumpRequest() []byte {
	return b.dump
}

// SetTimeout sets connect time out and read-write time out for BeegoRequest.
func (b *BeegoHttpRequest) SetTimeout(connectTimeout, readWriteTimeout time.Duration) *BeegoHttpRequest {
	b.Setting.ConnectTimeout = connectTimeout
	b.Setting.ReadWriteTimeout = readWriteTimeout
	return b
}

// SetTLSClientConfig sets tls connection configurations if visiting https url.
func (b *BeegoHttpRequest) SetTLSClientConfig(config *tls.Config) *BeegoHttpRequest {
	b.Setting.TlsClientConfig = config
	return b
}

// Header add header item string in request.
func (b *BeegoHttpRequest) Header(key, value string) *BeegoHttpRequest {
	b.Req.Header.Set(key, value)
	return b
}

// SetHost Set HOST
func (b *BeegoHttpRequest) SetHost(host string) *BeegoHttpRequest {
	b.Req.Host = host
	return b
}

// SetProtocolVersion Set the protocol version for incoming requests.
// Client requests always use HTTP/1.1.
func (b *BeegoHttpRequest) SetProtocolVersion(vers string) *BeegoHttpRequest {
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
func (b *BeegoHttpRequest) SetCookie(cookie *http.Cookie) *BeegoHttpRequest {
	b.Req.Header.Add("Cookie", cookie.String())
	return b
}

// SetTransport Set transport to
func (b *BeegoHttpRequest) SetTransport(transport http.RoundTripper) *BeegoHttpRequest {
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
func (b *BeegoHttpRequest) SetProxy(proxy func(*http.Request) (*url.URL, error)) *BeegoHttpRequest {
	b.Setting.Proxy = proxy
	return b
}

// Param adds query param in to request.
// params build query string as ?key1=value1&key2=value2...
func (b *BeegoHttpRequest) Param(key, value string) *BeegoHttpRequest {
	b.params[key] = value
	return b
}

func (b *BeegoHttpRequest) PostFile(formname, filename string) *BeegoHttpRequest {
	b.files[formname] = filename
	return b
}

// Body adds request raw body.
// it supports string and []byte.
func (b *BeegoHttpRequest) Body(data interface{}) *BeegoHttpRequest {
	switch t := data.(type) {
	case string:
		bf := bytes.NewBufferString(t)
		b.Req.Body = ioutil.NopCloser(bf)
		b.Req.ContentLength = int64(len(t))
	case []byte:
		bf := bytes.NewBuffer(t)
		b.Req.Body = ioutil.NopCloser(bf)
		b.Req.ContentLength = int64(len(t))
	}
	return b
}

// JsonBody adds request raw body encoding by JSON.
func (b *BeegoHttpRequest) JsonBody(obj interface{}) (*BeegoHttpRequest, error) {
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

func (b *BeegoHttpRequest) buildUrl(paramBody string) {
	// build GET url with query string
	if b.Req.Method == "GET" && len(paramBody) > 0 {
		if strings.Index(b.url, "?") != -1 {
			b.url += "&" + paramBody
		} else {
			b.url = b.url + "?" + paramBody
		}
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
					//iocopy
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
			b.Header("Content-Type", bodyWriter.FormDataContentType())
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

func (b *BeegoHttpRequest) getResponse() (*http.Response, error) {
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

func (b *BeegoHttpRequest) SendOut() (*http.Response, error) {
	var paramBody string
	if len(b.params) > 0 {
		var buf bytes.Buffer
		for k, v := range b.params {
			buf.WriteString(url.QueryEscape(k))
			buf.WriteByte('=')
			buf.WriteString(url.QueryEscape(v))
			buf.WriteByte('&')
		}
		paramBody = buf.String()
		paramBody = paramBody[0 : len(paramBody)-1]
	}

	b.buildUrl(paramBody)
	url, err := url.Parse(b.url)
	if err != nil {
		return nil, err
	}

	b.Req.URL = url

	trans := b.Setting.Transport

	if trans == nil {
		// create default transport
		trans = &http.Transport{
			TLSClientConfig: b.Setting.TlsClientConfig,
			Proxy:           b.Setting.Proxy,
			Dial:            TimeoutDialer(b.Setting.ConnectTimeout, b.Setting.ReadWriteTimeout),
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
			if t.Dial == nil {
				t.Dial = TimeoutDialer(b.Setting.ConnectTimeout, b.Setting.ReadWriteTimeout)
			}
		}
	}

	var jar http.CookieJar = nil
	if b.Setting.EnableCookie {
		if defaultCookieJar == nil {
			createDefaultCookie()
		}
		jar = defaultCookieJar
	}

	client := &http.Client{
		Transport: trans,
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
func (b *BeegoHttpRequest) String() (string, error) {
	data, err := b.Bytes()
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// Bytes returns the body []byte in response.
// it calls Response inner.
func (b *BeegoHttpRequest) Bytes() ([]byte, error) {
	if b.body != nil {
		return b.body, nil
	}
	resp, err := b.getResponse()
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
func (b *BeegoHttpRequest) ToFile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	resp, err := b.getResponse()
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
func (b *BeegoHttpRequest) ToJson(v interface{}) error {
	data, err := b.Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// ToXml returns the map that marshals from the body bytes as xml in response .
// it calls Response inner.
func (b *BeegoHttpRequest) ToXml(v interface{}) error {
	data, err := b.Bytes()
	if err != nil {
		return err
	}
	return xml.Unmarshal(data, v)
}

// Response executes request client gets response mannually.
func (b *BeegoHttpRequest) Response() (*http.Response, error) {
	return b.getResponse()
}

// TimeoutDialer returns functions of connection dialer with timeout settings for http.Transport Dial field.
func TimeoutDialer(cTimeout time.Duration, rwTimeout time.Duration) func(net, addr string) (c net.Conn, err error) {
	return func(netw, addr string) (net.Conn, error) {
		conn, err := net.DialTimeout(netw, addr, cTimeout)
		if err != nil {
			return nil, err
		}
		conn.SetDeadline(time.Now().Add(rwTimeout))
		return conn, nil
	}
}
