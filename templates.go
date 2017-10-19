package main

import (
	"html/template"
	"path/filepath"
	"strings"

	// "bytes"
	// "fmt"
	// "log"
)

const TEMPLATE_DIR string = "web/templates/"
const TEMPLATE_LAYOUT_DIR string = "web/templates/layout/"
const TEMPLATE_LAYOUT string = "web/templates/layout/base.html"

func loadTemplates(dir string) (map[string]*template.Template, error) {
	baseTemplate := template.New("base")
	baseTemplate, err := baseTemplate.ParseFiles(TEMPLATE_LAYOUT)
	if err != nil {
		return nil, err
	}

	layoutFiles, err := filepath.Glob(TEMPLATE_LAYOUT_DIR + "*.html")
	if err != nil {
		return nil, err
	}

	tempFiles, err := filepath.Glob(TEMPLATE_DIR + "*.html")
	if err != nil {
		return nil, err
	}

	m := make(map[string]*template.Template)
	for _, f := range tempFiles {
		name := filepath.Base(f)
		name = strings.Replace(name, ".html", "", -1)
		t, err := baseTemplate.Clone()
		if err != nil {
			return nil, err
		}
		files := append(layoutFiles, f)
		tmpl, err := t.ParseFiles(files...)
		if err != nil {
			return nil, err
		}
	 	if err != nil {
			return nil, err
		}
	 	m[name] = tmpl
	}
	return m, err
}
