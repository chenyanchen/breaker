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
	_interface := flag.String("interface", "", "(optional) The interfaces name")
	output := flag.String("output", "", "The output file name")
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
				var _type string
				switch v := field.Type.(type) {
				case *ast.SelectorExpr:
					_type = v.X.(*ast.Ident).Name + "." + v.Sel.Name
				case *ast.ArrayType:
					_type = "[]" + v.Elt.(*ast.Ident).Name
				case *ast.StarExpr:
					switch x := v.X.(type) {
					case *ast.Ident:
						paramType := x.Name
						_type = "*" + paramType
						if strings.Contains(paramType, ".") { // imported type
							break
						}
						paramIdent, ok := v.X.(*ast.Ident)
						if !ok {
							break
						}
						if paramIdent.Obj == nil {
							_type = "*" + packageName + "." + paramType // local type
							break
						}
						paramTypeSpec, ok := paramIdent.Obj.Decl.(*ast.TypeSpec)
						if !ok {
							break
						}
						if _, ok := paramTypeSpec.Type.(*ast.StructType); !ok {
							break
						}
						_type = "*" + packageName + "." + paramType // local type
					case *ast.SelectorExpr:
						_type = "*" + x.X.(*ast.Ident).Name + "." + x.Sel.Name
					}
				case *ast.Ellipsis:
					selectorExpr, ok := v.Elt.(*ast.SelectorExpr)
					if !ok {
						break
					}
					_type = "..." + selectorExpr.X.(*ast.Ident).Name + "." + selectorExpr.Sel.Name
				default:
					log.Fatalf("unsupported param type: %T", field.Type)
				}
				params = append(params, Param{Name: paramName, Type: _type})
			}
			for i, field := range fieldList.Results.List {
				paramName := "result" + strconv.Itoa(i)
				if len(field.Names) > 0 {
					paramName = field.Names[0].Name
				}
				var _type string
				switch v := field.Type.(type) {
				case *ast.Ident:
					_type = v.Name
				case *ast.ArrayType:
					_type = "[]" + v.Elt.(*ast.Ident).Name
				case *ast.StarExpr:
					switch x := v.X.(type) {
					case *ast.Ident:
						paramType := x.Name
						_type = "*" + paramType
						if strings.Contains(paramType, ".") { // imported type
							break
						}
						paramIdent, ok := v.X.(*ast.Ident)
						if !ok {
							break
						}
						if paramIdent.Obj == nil {
							_type = "*" + packageName + "." + paramType // local type
							break
						}
						paramTypeSpec, ok := paramIdent.Obj.Decl.(*ast.TypeSpec)
						if !ok {
							break
						}
						if _, ok := paramTypeSpec.Type.(*ast.StructType); !ok {
							break
						}
						_type = "*" + packageName + "." + paramType // local type
					case *ast.SelectorExpr:
						_type = "*" + x.X.(*ast.Ident).Name + "." + x.Sel.Name
					}
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
	Name string
	Type string
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
