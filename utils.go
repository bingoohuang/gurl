package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bingoohuang/gg/pkg/fla9"
	"github.com/bingoohuang/gg/pkg/ss"
)

func EnvBool(name string) bool {
	value := os.Getenv(name)
	return ss.AnyOfFold(value, "y", "yes", "1", "on", "true", "t")
}

func inSlice(str string, l []string) bool {
	for i := range l {
		if l[i] == str {
			return true
		}
	}
	return false
}

// FormatBytes Convert bytes to human-readable string. Like a 2 MB, 64.2 KB, 52 B
func FormatBytes(i int64) (result string) {
	switch {
	case i > (1024 * 1024 * 1024 * 1024):
		result = fmt.Sprintf("%#.02f TB", float64(i)/1024/1024/1024/1024)
	case i > (1024 * 1024 * 1024):
		result = fmt.Sprintf("%#.02f GB", float64(i)/1024/1024/1024)
	case i > (1024 * 1024):
		result = fmt.Sprintf("%#.02f MB", float64(i)/1024/1024)
	case i > 1024:
		result = fmt.Sprintf("%#.02f KB", float64(i)/1024)
	default:
		result = fmt.Sprintf("%d B", i)
	}
	result = strings.Trim(result, " ")
	return
}

func flagEnv(v *[]string, name, value, usage, envName string) {
	if value == "" {
		value = os.Getenv(envName)
	}
	var defaultValue []string
	if value != "" {
		defaultValue = []string{value}
	}

	fla9.StringsVar(v, name, defaultValue, usage)
}

func flagEnvVar(p *string, name, value, usage, envName string) {
	if value == "" {
		value = os.Getenv(envName)
	}
	fla9.StringVar(p, name, value, usage)
}
