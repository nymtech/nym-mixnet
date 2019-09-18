// Public Domain (-) 2010-2014 The Golly Authors.
// See the Golly UNLICENSE file for details.

package log

import (
	"golang.org/x/crypto/ssh/terminal"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var funcMap = template.FuncMap{
	"basepath": func(p string) string {
		return filepath.Base(p)
	},
	"color": func(style string) string {
		return ""
	},
	"json": func(v interface{}) string {
		out, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(out)
	},
	"jsonindent": func(v interface{}, indent string) string {
		out, err := json.MarshalIndent(v, "", indent)
		if err != nil {
			return ""
		}
		return string(out)
	},
	"lower": func(s string) string {
		return strings.ToLower(s)
	},
	"title": func(s string) string {
		return strings.ToTitle(s)
	},
	"upper": func(s string) string {
		return strings.ToUpper(s)
	},
}

var colorFuncMap = template.FuncMap{}

type Formatter interface {
	Write(*Entry, io.Writer) error
}

func JSONFormatter(pretty bool) Formatter {
	return &jsonFormatter{pretty}
}

type jsonFormatter struct {
	pretty bool
}

func (f *jsonFormatter) Write(e *Entry, w io.Writer) error {
	var (
		err error
		out []byte
	)
	if f.pretty {
		out, err = json.MarshalIndent(e, "", "  ")
	} else {
		out, err = json.Marshal(e)
	}
	if err != nil {
		return err
	}
	_, err = w.Write(append(out, '\xff', '\n'))
	return err
}

// SupportsColor inspects the file descriptor to figure out if it's attached to
// a terminal, and if so, returns true for color support. Example usage:
//
//     useColor := log.SupportsColor(os.Stdout)
func SupportsColor(f *os.File) bool {
	return terminal.IsTerminal(int(f.Fd()))
}

func TemplateFormatter(tmpl string, color bool, funcs template.FuncMap) (Formatter, error) {
	if funcs == nil || len(funcs) == 0 {
		if color {
			funcs = colorFuncMap
		} else {
			funcs = funcMap
		}
	} else {
		var base template.FuncMap
		if color {
			base = colorFuncMap
		} else {
			base = funcMap
		}
		for k, v := range base {
			if funcs[k] == nil {
				funcs[k] = v
			}
		}
	}
	indented := strings.Join(strings.Split(strings.TrimSpace(tmpl), "\n"), "\n    ")
	description := "log.formatter\n\n    " + indented + "\n\n"
	t, err := template.New(description).Funcs(funcs).Parse(tmpl)
	if err != nil {
		return nil, err
	}
	return &templateFormatter{
		template: t,
	}, nil
}

type templateFormatter struct {
	template *template.Template
}

func (f *templateFormatter) Write(e *Entry, w io.Writer) error {
	return f.template.Execute(w, e)
}
