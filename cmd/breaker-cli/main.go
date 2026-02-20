package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"go/types"
	"io"
	"log"
	"os"
	"path/filepath"
	"text/template"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/imports"
	"mvdan.cc/gofumpt/format"
)

func main() {
	_package := flag.String("package", "", "The packages named by the import paths")
	_interface := flag.String("interface", "", "The interface name")
	output := flag.String("output", "", "The output file name, default to stdout")
	flag.Parse()

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes,
	}
	file, err := packages.Load(cfg, *_package)
	if err != nil {
		log.Fatalf("load packages: %v", err)
	}
	if packages.PrintErrors(file) > 0 {
		log.Fatalf("load packages failed")
	}
	//
	if len(file) == 0 {
		log.Fatalf("no packages found")
	}

	packageName := file[0].Name

	// Extract the interface from the package scope using go/types.
	obj := file[0].Types.Scope().Lookup(*_interface)
	if obj == nil {
		log.Fatalf("interface %q not found in package %s", *_interface, packageName)
	}
	iface, ok := obj.Type().Underlying().(*types.Interface)
	if !ok {
		log.Fatalf("%q is not an interface", *_interface)
	}

	qualifier := func(p *types.Package) string { return p.Name() }

	var methods []Method
	for i := range iface.NumMethods() {
		m := iface.Method(i)
		sig := m.Type().(*types.Signature)

		params := extractParams(sig.Params(), sig.Variadic(), qualifier)
		returns := extractResults(sig.Results(), qualifier)

		methods = append(methods, Method{Name: m.Name(), Params: params, Results: returns})
	}

	// implement Interface
	implementation := implementInterface(Interface{Name: *_interface, Methods: methods})

	// generate code
	pkg := Package{
		Name:       packageName,
		ImportPath: *_package,
		Structs:    []Struct{implementation},
	}

	reader, err := generate(context.Background(), pkg, *output)
	if err != nil {
		log.Fatalf("generate: %v", err)
	}

	writer := os.Stdout
	if *output != "" {
		// TODO: Create the output directory if it does not exist.
		if dir := filepath.Dir(*output); dir != "." {
			if err = os.MkdirAll(dir, os.ModePerm); err != nil {
				log.Fatalf("mkdir: %v", err)
			}
		}
		writer, err = os.Create(*output)
		if err != nil {
			log.Fatalf("create file: %v", err)
		}
		defer writer.Close()
	}
	io.Copy(writer, reader)
}

func extractParams(tuple *types.Tuple, variadic bool, qualifier types.Qualifier) []Param {
	var params []Param
	for i := range tuple.Len() {
		v := tuple.At(i)
		name := v.Name()
		if name == "" {
			name = fmt.Sprintf("param%d", i)
		}

		isVariadic := variadic && i == tuple.Len()-1
		var typStr string
		if isVariadic {
			// For variadic params, the type is a *types.Slice; extract the element type.
			elem := v.Type().(*types.Slice).Elem()
			typStr = "..." + types.TypeString(elem, qualifier)
		} else {
			typStr = types.TypeString(v.Type(), qualifier)
		}

		params = append(params, Param{Name: name, Type: typStr, Ellipsis: isVariadic})
	}
	return params
}

func extractResults(tuple *types.Tuple, qualifier types.Qualifier) []Param {
	var results []Param
	for i := range tuple.Len() {
		v := tuple.At(i)
		name := v.Name()
		if name == "" {
			name = fmt.Sprintf("result%d", i)
		}
		results = append(results, Param{
			Name: name,
			Type: types.TypeString(v.Type(), qualifier),
		})
	}
	return results
}

func parseBreakerTemplate() (*template.Template, error) {
	tmpl, err := template.New("breaker.gohtml").Parse(breakerTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}
	return tmpl, nil
}

func generate(ctx context.Context, pkg Package, filename string) (io.Reader, error) {
	tmpl, err := parseBreakerTemplate()
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	if err = tmpl.Execute(buf, pkg); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	data, err := imports.Process(filename, buf.Bytes(), nil)
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
	Name       string
	ImportPath string
	Structs    []Struct
}

type Struct struct {
	Name        string
	Implemented Interface
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
	Name     string
	Type     string
	Ellipsis bool
}

func implementInterface(iface Interface) Struct {
	return Struct{
		Name:        iface.Name + "Breaker",
		Implemented: iface,
	}
}

