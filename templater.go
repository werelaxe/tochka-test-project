package main

import (
	"html/template"
	"io/ioutil"
	"path"
	"strings"
)

type Templater struct {
	templatesPath string
	templates     map[string]*template.Template
}

func (templater *Templater) Init(templatesPath string) {
	templater.templates = make(map[string]*template.Template)
	templater.templatesPath = templatesPath
	templateFiles, err := ioutil.ReadDir(templatesPath)
	if err != nil {
		panic("templater initialization error: " + err.Error())
	}
	for _, templateFile := range templateFiles {
		if !strings.HasSuffix(templateFile.Name(), ".html") {
			continue
		}
		templateFilename := path.Join(templatesPath, templateFile.Name())
		tmpl, err := template.ParseFiles(templateFilename)
		if err != nil {
			panic("templater initialization error: " + err.Error())
		}
		templater.templates[templateFilename[0:len(templateFilename)-5]] = tmpl
	}
}

func (templater *Templater) GetTemplate(name string) *template.Template {
	tmpl, ok := templater.templates[path.Join(templater.templatesPath, name)]
	if !ok {
		panic("no such template '" + name + "'")
	}
	return tmpl
}
