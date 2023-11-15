package main

import (
	"errors"
	"io"
	"os"
	"regexp"

	"github.com/bingoohuang/gg/pkg/iox"
	"github.com/bingoohuang/gg/pkg/osx/env"
	"github.com/bingoohuang/gg/pkg/vars"
	"github.com/bingoohuang/jj"
	"github.com/chzyer/readline"
)

var (
	valuer = NewValuer(env.Bool("INTERACTIVE", true))
	gen    = jj.NewGenContext(valuer)
)

func Eval(s string) (string, error) {
	var lines string
	for {
		blanks, left := eatBlanks(s)
		if len(blanks) > 0 {
			lines += blanks
		}
		genResult, i, err := gen.Process(left)
		if err != nil {
			return "", err
		}
		if i <= 0 {
			if s != "" {
				lines += s
			}
			break
		}

		lines += genResult.Out
		s = left[i:]

	}

	eval, err := vars.ParseExpr(lines).Eval(valuer)
	if err != nil {
		return "", err
	}
	return vars.ToString(eval), nil
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
	InteractiveMode bool
}

func NewValuer(interactiveMode bool) *Valuer {
	return &Valuer{
		Map:             make(map[string]interface{}),
		GenContext:      jj.NewGen(),
		InteractiveMode: interactiveMode,
	}
}

var cacheSuffix = regexp.MustCompile(`^(.+)_\d+`)

func (v *Valuer) ClearCache() {
	v.Map = make(map[string]interface{})
}

func (v *Valuer) Value(name, params, expr string) (any, error) {
	pureName := name
	subs := cacheSuffix.FindStringSubmatch(name)
	if len(subs) > 0 {
		pureName = subs[1]
		if x, ok := v.Map[name]; ok {
			return x, nil
		}
	}

	x, err := jj.DefaultGen.Value(pureName, params, expr)
	if err != nil {
		return nil, err
	}

	if x == expr && v.InteractiveMode { // 没有解析成功，进入命令行输入模式
		x = GetVar(name)
	}

	if len(subs) > 0 {
		v.Map[name] = x
	}

	return x, nil
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
}
