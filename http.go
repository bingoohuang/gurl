package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bingoohuang/gurl/httplib"
)

var defaultSetting = httplib.BeegoHttpSettings{
	ShowDebug:        true,
	UserAgent:        "gurl/" + version,
	ConnectTimeout:   60 * time.Second,
	ReadWriteTimeout: 60 * time.Second,
	Gzip:             true,
	DumpBody:         true,
}

func getHTTP(method string, url string, args []string) (r *httplib.BeegoHttpRequest) {
	r = httplib.NewRequest(url, method)
	r.Setting = defaultSetting
	r.Header("Accept-Encoding", "gzip, deflate")
	if *isjson {
		r.Header("Accept", "application/json")
		r.Header("Content-Type", "application/json")
	} else if form || method == "GET" {
		r.Header("Accept", "*/*")
	} else {
		r.Header("Accept", "application/json")
	}
	for i := range args {
		// Json raws
		if strs := strings.SplitN(args[i], ":=", 2); len(strs) == 2 {
			if v, fn, err := readFile(strs[1]); err != nil {
				log.Fatal("Read File", fn, err)
			} else if fn != "" {
				var j interface{}
				if err := json.Unmarshal(v, &j); err != nil {
					log.Fatal("Read from File", fn, "Unmarshal", err)
				}
				jsonmap[strs[0]] = j
			} else {
				jsonmap[strs[0]] = json.RawMessage(strs[1])
			}
			continue
		}
		// Headers
		if strs := strings.Split(args[i], ":"); len(strs) >= 2 {
			if strs[0] == "Host" {
				r.SetHost(strings.Join(strs[1:], ":"))
			}
			r.Header(strs[0], strings.Join(strs[1:], ":"))
			continue
		}
		// Params
		if strs := strings.SplitN(args[i], "=", 2); len(strs) == 2 {
			strs[1] = tryReadFile(strs[1])
			if form || method == "GET" {
				r.Param(strs[0], strs[1])
			} else {
				jsonmap[strs[0]] = strs[1]
			}
			continue
		}
		// files
		if strs := strings.SplitN(args[i], "@", 2); len(strs) == 2 {
			r.PostFile(strs[0], strs[1])
			continue
		}
	}
	if !form && len(jsonmap) > 0 {
		if _, err := r.JsonBody(jsonmap); err != nil {
			log.Fatal("fail to marshal JSON: ", err)
		}
	}
	return
}

func tryReadFile(s string) string {
	if v, _, err := readFile(s); err != nil {
		log.Fatal("Read File", s, err)
		return ""
	} else {
		return string(v)
	}
}

func readFile(s string) (data []byte, fn string, e error) {
	if !strings.HasPrefix(s, "@") {
		return []byte(s), "", nil
	}

	s = strings.TrimLeft(s, "@")
	f, err := os.Open(s)
	if err != nil {
		return nil, s, err
	}
	content, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, s, err
	}
	return content, s, nil
}

func formatResponseBody(res *http.Response, httpreq *httplib.BeegoHttpRequest, pretty bool) string {
	body, err := httpreq.Bytes()
	if err != nil {
		log.Fatalln("can't get the url", err)
	}
	fmt.Println("")
	match := contentJsonRegex.MatchString(res.Header.Get("Content-Type"))
	if pretty && match {
		var output bytes.Buffer
		err := json.Indent(&output, body, "", "  ")
		if err != nil {
			log.Fatal("Response JSON Indent: ", err)
		}

		return output.String()
	}

	return string(body)
}
