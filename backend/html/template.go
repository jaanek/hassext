package html

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"text/template"
	"time"

	"golang.org/x/exp/maps"
)

func fdate(t time.Time) string {
	return t.Format("02.01.2006")
}

func ftime(t time.Time) string {
	return t.Format("15:04")
}

func NewFuncMap(pfm template.FuncMap) template.FuncMap {
	var fm = template.FuncMap{
		"fdate": fdate,
		"ftime": ftime,
		"add":   add,
		"div":   divide,
	}
	if pfm != nil {
		maps.Copy(fm, pfm)
	}
	return fm
}

//go:embed templates/*.html
var Templates embed.FS

func GenerateFS(folder fs.FS, dir, file string, pfm template.FuncMap, data interface{}) (string, error) {
	var fm = NewFuncMap(pfm)
	tpl, err := template.New("").Funcs(fm).ParseFS(folder, path.Join(dir, file))
	if err != nil {
		return "", fmt.Errorf("file not found: %v", err)
	}
	bytes := &bytes.Buffer{}
	err = tpl.ExecuteTemplate(bytes, file, data)
	if err != nil {
		return "", fmt.Errorf("template execution error: %v", err)
	}
	return string(bytes.Bytes()), nil
}

func Generate(tmplStr string, pfm template.FuncMap, data interface{}) (string, error) {
	var fm = NewFuncMap(pfm)

	// Create a new template & parse the template string
	tpl, err := template.New("").Funcs(fm).Parse(tmplStr)
	if err != nil {
		return "", err
	}

	// Execute the template with the data
	bytes := &bytes.Buffer{}
	err = tpl.Execute(bytes, data)
	if err != nil {
		return "", err
	}
	return string(bytes.Bytes()), nil
}

type GeneralEmail struct {
	Body string
}
