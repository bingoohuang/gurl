package main

import (
	"strings"

	"github.com/bingoohuang/gg/pkg/rest"
	"github.com/bingoohuang/gg/pkg/ss"
)

var methodList = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

func filter(args []string) []string {
	var filteredArgs []string
	methodFoundInArgs := false
	defaultSchema := rest.WithDefaultScheme(ss.If(caFile != "", "https", "http"))

	for _, arg := range args {
		if arg == "version" {
			ver = true
			continue
		}

		if ss.HasPrefix(arg, "http:", "https:") {
			*Urls = append(*Urls, arg)
			continue
		}

		if subs := keyReq.FindStringSubmatch(arg); len(subs) > 0 && subs[1] != "" {
			filteredArgs = append(filteredArgs, arg)
			continue
		}

		if inSlice(strings.ToUpper(arg), methodList) {
			*method = strings.ToUpper(arg)
			methodFoundInArgs = true
		} else if addr := rest.FixURI(arg, defaultSchema); addr.OK() && strings.ContainsAny(arg, ":/") {
			*Urls = append(*Urls, addr.Data.String())
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
		}
	}

	if !methodFoundInArgs && *method == "GET" && body != "" {
		*method = "POST"
	}

	return args
}
