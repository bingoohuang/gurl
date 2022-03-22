package main

import (
	"log"
	"net/url"
	"regexp"
	"strings"
)

var methodList = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

func filter(args []string) []string {
	var filteredArgs []string
	methodFoundInArgs := false

	for _, arg := range args {
		if inSlice(strings.ToUpper(arg), methodList) {
			*method = strings.ToUpper(arg)
			methodFoundInArgs = true
		} else if urlAddr, err := FixURI(arg); err == nil && strings.ContainsAny(arg, ":/") {
			*Urls = append(*Urls, urlAddr)
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
				submatch := keyReq.FindStringSubmatch(v)
				if len(submatch) == 0 {
					continue
				}

				// defaults to either GET (with no request data) or POST (with request data).
				switch _, op, _ := submatch[1], submatch[2], submatch[3]; op {
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
	if len(*Urls) == 0 {
		log.Fatal("Miss the URL")
	}
	return args
}

var reScheme = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+-.]*://`)

func FixURI(uri string) (string, error) {
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
