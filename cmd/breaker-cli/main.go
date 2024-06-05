package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"go/ast"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo}
	file, err := packages.Load(cfg, *_package)
	if err != nil {
		log.Fatalf("load packages: %v", err)
	}
	//
	if len(file) == 0 {
		log.Fatalf("no packages found")
	}

	packageName := file[0].Name

	// Extract the interface
	var interfaces []Interface
	for ident := range file[0].TypesInfo.Defs {
		if ident.Name != *_interface {
			continue
		}
		if ident.Obj == nil {
			continue
		}
		if ident.Obj.Kind != ast.Typ {
			continue
		}
		typeSpec, ok := ident.Obj.Decl.(*ast.TypeSpec)
		if !ok {
			continue
		}
		interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
		if !ok {
			continue
		}

		var methods []Method
		for _, method := range interfaceType.Methods.List {
			methodName := method.Names[0].Name
			var params, returns []Param

			fieldList, ok := method.Type.(*ast.FuncType)
			if !ok {
				continue
			}

			for i, field := range fieldList.Params.List {
				paramName := "param" + strconv.Itoa(i)
				if len(field.Names) > 0 {
					paramName = field.Names[0].Name
				}
				ellipsis := false
				var _type string
				switch v := field.Type.(type) {
				case *ast.Ident:
					_type = identName(packageName, v)
				case *ast.StarExpr:
					_type = starExprName(packageName, v)
				case *ast.SelectorExpr:
					_type = selectorExprName(packageName, v)
				case *ast.ArrayType:
					_type = arrayTypeName(packageName, v)
				case *ast.MapType:
					_type = mapTypeName(packageName, v)
				case *ast.Ellipsis:
					ellipsis = true
					_type = ellipsisName(packageName, v)
				default:
					log.Fatalf("unsupported param type: %T", field.Type)
				}
				params = append(params, Param{Name: paramName, Type: _type, Ellipsis: ellipsis})
			}
			for i, field := range fieldList.Results.List {
				paramName := "result" + strconv.Itoa(i)
				if len(field.Names) > 0 {
					paramName = field.Names[0].Name
				}
				var _type string
				switch v := field.Type.(type) {
				case *ast.Ident:
					_type = identName(packageName, v)
				case *ast.StarExpr:
					_type = starExprName(packageName, v)
				case *ast.SelectorExpr:
					_type = selectorExprName(packageName, v)
				case *ast.ArrayType:
					_type = arrayTypeName(packageName, v)
				case *ast.MapType:
					_type = mapTypeName(packageName, v)
				default:
					log.Fatalf("unsupported return type: %T", field.Type)
				}
				returns = append(returns, Param{Name: paramName, Type: _type})
			}

			methods = append(methods, Method{Name: methodName, Params: params, Results: returns})
		}

		interfaces = append(interfaces, Interface{Name: ident.Name, Methods: methods})
	}

	if len(interfaces) == 0 {
		log.Fatalf("no interface found")
	}

	// implement Interface
	implementation := implementInterface(interfaces[0])

	// generate code
	pkg := Package{
		Name:    packageName,
		Structs: []Struct{implementation},
	}

	reader, err := generate(context.Background(), pkg)
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

func identName(pkg string, t *ast.Ident) string {
	// TODO: do you have a better way to handle basic types?

	// basic types
	switch t.Name {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64",
		"string", "bool", "byte", "rune", "error":
		return t.Name
	}

	// package types
	return pkg + "." + t.Name
}

func ellipsisName(pkg string, t *ast.Ellipsis) string {
	var name string
	switch x := t.Elt.(type) {
	case *ast.Ident:
		name = identName(pkg, x)
	case *ast.SelectorExpr:
		name = selectorExprName(pkg, x)
	case *ast.StarExpr:
		name = starExprName(pkg, x)
	case *ast.ArrayType:
		name = arrayTypeName(pkg, x)
	case *ast.MapType:
		name = mapTypeName(pkg, x)
	default:
		panic(fmt.Sprintf("unsupported ellipsis type: %T", x))
	}
	return "..." + name
}

func selectorExprName(pkg string, t *ast.SelectorExpr) string {
	switch x := t.X.(type) {
	case *ast.Ident:
		return x.Name + "." + t.Sel.Name
	default:
		panic(fmt.Sprintf("unsupported selector expr: %T", t))
	}
}

func starExprName(pkg string, t *ast.StarExpr) string {
	var name string
	switch x := t.X.(type) {
	case *ast.Ident:
		name = identName(pkg, x)
	case *ast.ArrayType:
		name = arrayTypeName(pkg, x)
	case *ast.MapType:
		name = mapTypeName(pkg, x)
	case *ast.SelectorExpr:
		name = selectorExprName(pkg, x)
	default:
		panic(fmt.Sprintf("unsupported star expr: %T", t))
	}
	return "*" + name
}

func arrayTypeName(pkg string, t *ast.ArrayType) string {
	var name string
	switch elt := t.Elt.(type) {
	case *ast.Ident:
		name = identName(pkg, elt)
	case *ast.SelectorExpr:
		name = selectorExprName(pkg, elt)
	case *ast.StarExpr:
		name = starExprName(pkg, elt)
	case *ast.ArrayType:
		name = arrayTypeName(pkg, elt)
	case *ast.MapType:
		name = mapTypeName(pkg, elt)
	default:
		panic(fmt.Sprintf("unsupported array type: %T", elt))
	}
	return "[]" + name
}

func mapTypeName(pkg string, t *ast.MapType) string {
	var key, value string
	switch k := t.Key.(type) {
	case *ast.Ident:
		key = identName(pkg, k)
	case *ast.StarExpr:
		key = starExprName(pkg, k)
	default:
		panic(fmt.Sprintf("unsupported map key type: %T", k))
	}

	switch v := t.Value.(type) {
	case *ast.Ident:
		value = identName(pkg, v)
	case *ast.SelectorExpr:
		value = selectorExprName(pkg, v)
	case *ast.StarExpr:
		value = starExprName(pkg, v)
	case *ast.ArrayType:
		value = arrayTypeName(pkg, v)
	case *ast.MapType:
		value = mapTypeName(pkg, v)
	default:
		panic(fmt.Sprintf("unsupported map value type: %T", v))
	}

	return fmt.Sprintf("map[%s]%s", key, value)
}

func generate(ctx context.Context, pkg Package) (io.Reader, error) {
	tmpl, err := template.New("breaker.gohtml").Parse(breakerTemplate)
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

func toLowerCamelCase(s string) string {
	firstChar := strings.ToLower(s[:1])
	return firstChar + s[1:]
}
