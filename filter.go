package main

import (
	"net/url"
	"regexp"
	"strings"
)

var methodList = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

func filter(args []string) []string {
	var filteredArgs []string
	methodFoundInArgs := false

	for _, arg := range args {
		if arg == "version" {
			ver = true
			continue
		}

		if subs := keyReq.FindStringSubmatch(arg); len(subs) > 0 && subs[1] != "" {
			filteredArgs = append(filteredArgs, arg)
			continue
		}

		if inSlice(strings.ToUpper(arg), methodList) {
			*method = strings.ToUpper(arg)
			methodFoundInArgs = true
		} else if addr, err := FixURI(arg, caFile); err == nil && strings.ContainsAny(arg, ":/") {
			*Urls = append(*Urls, addr)
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}
	args = filteredArgs

	if !methodFoundInArgs && *method == "GET" {
		if len(uploadFiles) > 0 {
			*method = "POST"
		} else if len(args) > 0 {
			for _, v := range args[1:] {
				subs := keyReq.FindStringSubmatch(v)
				if len(subs) == 0 {
					continue
				}

				// defaults to either GET (with no request data) or POST (with request data).
				switch _, op, _ := subs[1], subs[2], subs[3]; op {
				case ":=": // Json raws
					*method = "POST"
				case "==": // Queries
				case "=": // Params
					*method = "POST"
				case ":": // Headers
				case "@": // files
					*method = "POST"
				}
				if *method == "POST" {
					break
				}
			}
		} else if body != "" {
			*method = "POST"
		}
	}

	return args
}

var reScheme = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+-.]*://`)

func FixURI(uri string, caFile string) (string, error) {
	if uri == ":" {
		uri = ":80"
	}

	defaultScheme, defaultHost := "http", "localhost"
	// ex) :8080/hello or /hello or :
	if strings.HasPrefix(uri, ":") || strings.HasPrefix(uri, "/") {
		uri = defaultHost + uri
	}

	if caFile != "" {
		defaultScheme = "https"
	}
	// ex) example.com/hello
	if !reScheme.MatchString(uri) {
		uri = defaultScheme + "://" + uri
	}

	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}

	u.Host = strings.TrimSuffix(u.Host, ":")
	if u.Path == "" {
		u.Path = "/"
	}

	return u.String(), nil
}
