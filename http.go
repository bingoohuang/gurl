package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/bingoohuang/gg/pkg/v"

	"github.com/bingoohuang/gurl/httplib"
)

var defaultSetting = httplib.Settings{
	ShowDebug:        true,
	UserAgent:        "gurl/" + v.AppVersion,
	ConnectTimeout:   60 * time.Second,
	ReadWriteTimeout: 60 * time.Second,
	Gzip:             true,
	DumpBody:         true,
}

var keyReq = regexp.MustCompile(`^([\d\w_.\-]+)(==|:=|=|:|@)(.*)`)

func getHTTP(method string, url string, args []string, timeout time.Duration) (r *httplib.Request) {
	r = httplib.NewRequest(url, method)
	r.DisableKeepAlives = disableKeepAlive
	r.Setting = defaultSetting
	r.Setting.ReadWriteTimeout = timeout
	r.Header("Accept-Encoding", "gzip, deflate")
	if *isjson {
		r.Header("Accept", "application/json")
		r.Header("Content-Type", "application/json")
	} else if form || method == "GET" {
		r.Header("Accept", "*/*")
	} else {
		r.Header("Accept", "application/json")
	}
	r.Header("Gurl-Date", time.Now().UTC().Format(http.TimeFormat))
	// https://httpie.io/docs#request-items
	// Item Type	Description
	// HTTP Headers Name:Value	Arbitrary HTTP header, e.g. X-API-Token:123
	// URL parameters name==value	Appends the given name/value pair as a querystring parameter to the URL. The == separator is used.
	// Data Fields field=value, field=@file.txt	Request data fields to be serialized as a JSON object (default), to be form-encoded (with --form, -f), or to be serialized as multipart/form-data (with --multipart)
	// Raw JSON fields field:=json	Useful when sending JSON and one or more fields need to be a Boolean, Number, nested Object, or an Array, e.g., meals:='["ham","spam"]' or pies:=[1,2,3] (note the quotes)
	// File upload fields field@/dir/file, field@file;type=mime	Only available with --form, -f and --multipart. For example screenshot@~/Pictures/img.png, or 'cv@cv.txt;type=text/markdown'. With --form, the presence of a file field results in a --multipart request
	for i := range args {
		submatch := keyReq.FindStringSubmatch(args[i])
		if len(submatch) == 0 {
			continue
		}

		switch k, op, v := submatch[1], submatch[2], submatch[3]; op {
		case ":=": // Json raws
			if v, fn, err := readFile(v); err != nil {
				log.Fatal("Read File", fn, err)
			} else if fn != "" {
				var j interface{}
				if err := json.Unmarshal(v, &j); err != nil {
					log.Fatal("Read from File", fn, "Unmarshal", err)
				}
				jsonmap[k] = j
			} else {
				jsonmap[k] = json.RawMessage(v)
			}
		case "==": // Queries
			r.Query(k, tryReadFile(v))
		case "=": // Params
			if v = tryReadFile(v); form || method == "GET" {
				r.Param(k, v)
			} else {
				jsonmap[k] = v
			}
		case ":": // Headers
			if k == "Host" {
				r.SetHost(v)
			} else {
				r.Header(k, v)
			}
		case "@": // files
			r.PostFile(k, v)
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
	v, _, err := readFile(s)
	if err != nil {
		log.Fatal("Read File", s, err)
	}

	return string(v)
}

func readFile(s string) (data []byte, fn string, e error) {
	if !strings.HasPrefix(s, "@") {
		return []byte(s), "", nil
	}

	s = strings.TrimPrefix(s, "@")
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

func formatResponseBody(r *httplib.Request, pretty, hasDevice bool) string {
	body, err := r.Bytes()
	if err != nil {
		log.Fatalln("can't get the url", err)
	}

	return formatBytes(body, pretty, hasDevice)
}

func formatBytes(body []byte, pretty, hasDevice bool) string {
	isJSON := json.Valid(body)
	if pretty && isJSON {
		var output bytes.Buffer
		if err := json.Indent(&output, body, "", "  "); err == nil {
			body = output.Bytes()
		}
	}

	if hasDevice {
		return ColorfulResponse(string(body), isJSON)
	}

	return string(body)
}
