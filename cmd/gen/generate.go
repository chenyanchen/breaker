package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"text/template"

	"golang.org/x/tools/imports"
	"mvdan.cc/gofumpt/format"
)

func generate(ctx context.Context, pkg Package) (io.Reader, error) {
	tmpl, err := template.ParseFiles("breaker.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	buf := &bytes.Buffer{}
	if err = tmpl.Execute(buf, pkg); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	data, err := imports.Process("", buf.Bytes(), nil)
	if err != nil {
		return nil, fmt.Errorf("goimports: %w", err)
	}

	data, err = format.Source(data, format.Options{})
	if err != nil {
		return nil, fmt.Errorf("gofumpt: %w", err)
	}

	buf.Reset()
	if _, err = buf.Write(data); err != nil {
		return nil, fmt.Errorf("write buffer: %w", err)
	}

	return buf, nil
}

type Package struct {
	Name    string
	Structs []Struct
}

type Struct struct {
	Name       string
	TypeParams []Param // generic type params
	Interface  Interface
}

type Interface struct {
	Name    string
	Methods []Method
}

type Method struct {
	Name    string
	Params  []Param
	Results []Param
}

type Param struct {
	Name string
	Type string
}
