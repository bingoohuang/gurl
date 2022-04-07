package main

import (
	"encoding/json"
	"github.com/bingoohuang/jj"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/bingoohuang/gg/pkg/iox"
	"github.com/bingoohuang/gg/pkg/v"
)

var defaultSetting = Settings{
	ShowDebug:      true,
	UserAgent:      "gurl/" + v.AppVersion,
	ConnectTimeout: 60 * time.Second,
	DumpBody:       true,
}

var keyReq = regexp.MustCompile(`^([\d\w_.\-]*)(==|:=|=|:|@)(.*)`)

func getHTTP(method string, url string, args []string, timeout time.Duration) (r *Request) {
	r = NewRequest(url, method)
	r.DisableKeepAlives = disableKeepAlive
	r.Setting = defaultSetting
	r.Setting.ConnectTimeout = timeout
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
		arg := args[i]
		subs := keyReq.FindStringSubmatch(arg)
		if len(subs) == 0 {
			continue
		}

		k, op, val := subs[1], subs[2], subs[3]
		if k == "" && op != "@" {
			log.Fatalf("Unsupported argument %s", arg)
		}

		switch op {
		case ":=": // Json raws
			if dat, fn, err := readFile(val); err != nil {
				log.Fatal("Read File", fn, err)
			} else if fn != "" {
				var j interface{}
				if err := json.Unmarshal(dat, &j); err != nil {
					log.Fatal("Read from File", fn, "Unmarshal", err)
				}
				jsonmap[k] = j
			} else {
				jsonmap[k] = json.RawMessage(dat)
			}
		case "==": // Queries
			r.Query(k, tryReadFile(val))
		case "=": // Params
			if val = tryReadFile(val); form || method == "GET" {
				r.Param(k, val)
			} else {
				jsonmap[k] = val
			}
		case ":": // Headers
			if k == "Host" {
				r.SetHost(val)
			} else {
				r.Header(k, val)
			}
		case "@": // files
			if k != "" {
				r.PostFile(k, val)
			} else {
				r.Body(arg)
			}
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
	dat, _, err := readFile(s)
	if err != nil {
		log.Fatal("Read File", s, err)
	}

	return string(dat)
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
	defer iox.Close(f)

	content, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, s, err
	}
	return content, s, nil
}

func formatResponseBody(r *Request, pretty, hasDevice bool) string {
	dat, err := r.Bytes()
	if err != nil {
		log.Fatalln("can't get the url", err)
	}

	return formatBytes(dat, pretty, hasDevice)
}

func formatBytes(body []byte, pretty, hasDevice bool) string {
	isJSON := json.Valid(body)

	if isJSON {
		if pretty {
			body = jj.Pretty(body, jj.DefaultOptions)
		} else {
			body = jj.Ugly(body)
		}
	}

	if hasDevice {
		return ColorfulResponse(string(body), isJSON)
	}

	return string(body)
}
