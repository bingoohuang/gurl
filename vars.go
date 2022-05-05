package main

import (
	"regexp"

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

var cacheSuffix = regexp.MustCompile(`^(.+)_cache(_\d+)?`)

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

	if len(subs) > 0 {
		v.Map[name] = x
	}
	return x
}
