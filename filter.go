package main

import (
	"log"
	"strings"
)

var methodList = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

func filter(args []string) []string {
	var i int
	if inSlice(strings.ToUpper(args[i]), methodList) {
		*method = strings.ToUpper(args[i])
		i++
	} else if len(args) > 0 && *method == "GET" {
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
	} else if *method == "GET" && body != "" {
		*method = "POST"
	}
	if len(args) <= i {
		log.Fatal("Miss the URL")
	}
	*URL = args[i]
	i++
	return args[i:]
}
