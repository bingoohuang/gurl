package main

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"log"
	"os"
	"strings"

	"github.com/bingoohuang/gg/pkg/fla9"
)

func surveyConfirm() {
	qs := []*survey.Question{{
		Name: "confirm",
		Prompt: &survey.Select{
			Message: "Please confirm your action:",
			Options: []string{"continue", "exit"},
			Default: "continue",
		},
	}}

	// the answers will be written to this struct
	answers := struct {
		Confirm string
	}{}

	// perform the questions
	if err := survey.Ask(qs, &answers); err != nil {
		log.Fatal(err)
	}

	switch answers.Confirm {
	case "exit":
		os.Exit(0)
	}
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

const EnvPrefix = "GURL_"

func flagEnv(v *[]string, name, value, usage string) {
	if value == "" {
		value = os.Getenv(EnvPrefix + strings.ToUpper(name))
	}
	var defaultValue []string
	if value != "" {
		defaultValue = []string{value}
	}

	fla9.StringsVar(v, name, defaultValue, usage)
}

func flagEnvVar(p *string, name, value, usage string) {
	if value == "" {
		value = os.Getenv(EnvPrefix + strings.ToUpper(name))
	}
	fla9.StringVar(p, name, value, usage)
}
