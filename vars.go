package main

import (
	"log"
	"regexp"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bingoohuang/gg/pkg/vars"
	"github.com/bingoohuang/jj"
)

var (
	valuer = NewValuer()
	gen    = jj.NewGenContext(valuer)
)

func Eval(s string) string {
	var lines string
	for {
		blanks, left := eatBlanks(s)
		if len(blanks) > 0 {
			lines += blanks
		}
		genResult, i := gen.Process(left)
		if i <= 0 {
			break
		}

		lines += genResult.Out
		s = left[i:]

	}

	if len(lines) > 0 {
		return lines
	}

	return vars.ToString(vars.ParseExpr(s).Eval(valuer))
}

func eatBlanks(s string) (blanks, left string) {
	for i, c := range s {
		if c == ' ' || c == '\r' || c == '\n' {
			blanks += string(c)
		} else {
			left = s[i:]
			break
		}
	}

	return
}

type Valuer struct {
	Map map[string]interface{}
}

func NewValuer() *Valuer {
	return &Valuer{
		Map: make(map[string]interface{}),
	}
}

func (v *Valuer) Register(fn string, f jj.SubstitutionFn) {
	jj.DefaultSubstituteFns.Register(fn, f)
}

var cacheSuffix = regexp.MustCompile(`^(.+)_\d+`)

func (v *Valuer) ClearCache() {
	v.Map = make(map[string]interface{})
}

func (v *Valuer) Value(name, params string) interface{} {
	pureName := name
	subs := cacheSuffix.FindStringSubmatch(name)
	if len(subs) > 0 {
		pureName = subs[1]
		if x, ok := v.Map[name]; ok {
			return x
		}
	}

	x := jj.DefaultGen.Value(pureName, params)
	if x == "" {
		x = surveyValue(name)
	}

	if len(subs) > 0 {
		v.Map[name] = x
	}
	return x
}

func surveyValue(name string) string {
	qs := []*survey.Question{{
		Name:     "value",
		Prompt:   &survey.Input{Message: "Input the value of " + name + ":"},
		Validate: survey.Required,
	}}

	// the answers will be written to this struct
	answers := struct {
		Value string // survey will match the question and field names
	}{}

	// perform the questions
	if err := survey.Ask(qs, &answers); err != nil {
		log.Fatal(err)
	}
	return answers.Value
}
