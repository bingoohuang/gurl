package main

import (
	"errors"
	"io"
	"os"
	"regexp"

	"github.com/bingoohuang/gg/pkg/iox"
	"github.com/bingoohuang/gg/pkg/vars"
	"github.com/bingoohuang/jj"
	"github.com/chzyer/readline"
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
			if s != "" {
				lines += s
			}
			break
		}

		lines += genResult.Out
		s = left[i:]

	}

	return vars.ToString(vars.ParseExpr(lines).Eval(valuer))
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
	*jj.GenContext
}

func NewValuer() *Valuer {
	return &Valuer{
		Map:        make(map[string]interface{}),
		GenContext: jj.NewGen(),
	}
}

var cacheSuffix = regexp.MustCompile(`^(.+)_\d+`)

func (v *Valuer) ClearCache() {
	v.Map = make(map[string]interface{})
}

func (v *Valuer) Value(name, params, expr string) interface{} {
	pureName := name
	subs := cacheSuffix.FindStringSubmatch(name)
	if len(subs) > 0 {
		pureName = subs[1]
		if x, ok := v.Map[name]; ok {
			return x
		}
	}

	x := jj.DefaultGen.Value(pureName, params, expr)
	if x == expr {
		x = GetVar(name)
	}

	if len(subs) > 0 {
		v.Map[name] = x
	}
	return x
}

func GetVar(name string) string {
	line, err := ReadLine(
		WithPrompt(name+": "),
		WithHistoryFile("/tmp/gurl-vars-"+name),
		WithTrimSuffix(true))
	if errors.Is(err, io.EOF) {
		os.Exit(0)
	}
	return line
}

type LineConfig struct {
	Prompt      string
	HistoryFile string
	Suffix      []string
	TrimSuffix  bool
}

type LineConfigFn func(config *LineConfig)

func WithTrimSuffix(trimSuffix bool) LineConfigFn {
	return func(c *LineConfig) {
		c.TrimSuffix = trimSuffix
	}
}

func WithPrompt(prompt string) LineConfigFn {
	return func(c *LineConfig) {
		c.Prompt = prompt
	}
}

func WithHistoryFile(historyFile string) LineConfigFn {
	return func(c *LineConfig) {
		c.HistoryFile = historyFile
	}
}

func ReadLine(fns ...LineConfigFn) (string, error) {
	c := &LineConfig{
		Prompt:      "> ",
		HistoryFile: "/tmp/line",
	}
	for _, fn := range fns {
		fn(c)
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:                 c.Prompt,
		HistoryFile:            c.HistoryFile,
		DisableAutoSaveHistory: true,
	})
	if err != nil {
		panic(err)
	}

	defer iox.Close(rl)

	line, err := rl.Readline()
	if err != nil {
		if errors.Is(err, readline.ErrInterrupt) {
			return "", io.EOF
		}
		return "", err
	}
	return line, nil

	return line, nil
}
