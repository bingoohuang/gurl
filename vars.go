package main

import (
	"github.com/bingoohuang/gg/pkg/vars"
	"github.com/bingoohuang/jj"
)

var (
	valuer = NewValuer()
	gen    = jj.NewGenContext(valuer)
)

func Eval(s string) string {
	if jj.Parse(s).IsJSON() {
		return gen.Gen(s)
	}

	return vars.ToString(vars.ParseExpr(s).Eval(valuer))
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

func (v *Valuer) Value(name, params string) interface{} {
	if x, ok := v.Map[name]; ok {
		return x
	}

	x := jj.DefaultGen.Value(name, params)
	v.Map[name] = x
	return x
}
