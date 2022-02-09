package main

import (
	"fmt"
	"strings"

	"github.com/bingoohuang/jj"
)

const (
	Gray = uint8(iota + 90)
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White

	EndColor = "\033[0m"
)

func Color(str string, color uint8) string {
	return fmt.Sprintf("%s%s%s", ColorStart(color), str, EndColor)
}

func ColorStart(color uint8) string {
	return fmt.Sprintf("\033[%dm", color)
}

func ColorfulRequest(str string) string {
	lines := strings.Split(str, "\n")
	if printOption&printReqHeader == printReqHeader {
		strs := strings.Split(lines[0], " ")
		strs[0] = Color(strs[0], Magenta)
		strs[1] = Color(strs[1], Cyan)
		strs[2] = Color(strs[2], Magenta)
		lines[0] = strings.Join(strs, " ")
	}
	for i, line := range lines[1:] {
		substr := strings.Split(line, ":")
		if len(substr) < 2 {
			continue
		}
		substr[0] = Color(substr[0], Gray)
		substr[1] = Color(strings.Join(substr[1:], ":"), Cyan)
		lines[i+1] = strings.Join(substr[:2], ":")
	}
	return strings.Join(lines, "\n")
}

func ColorfulResponse(str string, isJSON bool) string {
	if isJSON {
		return string(jj.Color([]byte(str), nil))
	}

	return ColorfulHTML(str)
}

func ColorfulHTML(str string) string {
	return Color(str, Green)
}
