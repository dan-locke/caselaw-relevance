package main

import (
	"html/template"
	"path/filepath"
	"strings"
)

func loadTemplates(layout, layoutDir, templateDir, dir string) (map[string]*template.Template, error) {
	baseTemplate := template.New("base")
	baseTemplate, err := baseTemplate.ParseFiles(layout)
	if err != nil {
		return nil, err
	}

	layoutFiles, err := filepath.Glob(layoutDir + "*.html")
	if err != nil {
		return nil, err
	}

	tempFiles, err := filepath.Glob(templateDir + "*.html")
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
