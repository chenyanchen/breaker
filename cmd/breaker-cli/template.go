package main

const breakerTemplate = `// Code generated by breaker-cli. DO NOT EDIT.
// breaker-cli: https://github.com/chenyanchen/breaker/tree/main/cmd/breaker-cli

{{$PackageName := .Name -}}
package {{$PackageName}}

{{range .Structs -}}
    {{$StructName := .Name -}}
    {{$InterfaceName := .Implemented.Name -}}
type {{$StructName}} struct {
	source  {{$PackageName}}.{{$InterfaceName}}
	breaker breaker.Breaker
}

func New{{$StructName}}(source {{$PackageName}}.{{$InterfaceName}}) *{{$StructName}} {
	return &{{$StructName}}{
		source:  source,
		breaker: breaker.NewGoogleBreaker(),
	}
}

{{range .Implemented.Methods -}}
    {{$ErrorReturn := "" -}}
func (b *{{$StructName}}) {{.Name}}({{range $i, $p := .Params}}{{.Name}} {{.Type}}, {{end}}) ({{range .Results}}{{.Type}}, {{end}}) {
	{{/* Find the error return */}}
	{{- range .Results}}{{if eq .Type "error"}}{{$ErrorReturn = .Name}}{{end}}{{end -}}

	{{/* Return directly if there are no error return */}}
	{{- if eq $ErrorReturn "" -}}
	return b.source.{{.Name}}({{range .Params}}{{if eq .Ellipsis true}}{{.Name}}...{{else}}{{.Name}},{{end}}{{end}})
	{{- else -}}
	var (
		{{- range .Results}}
		{{.Name}} {{.Type}}
		{{- end}}
	)
	{{$ErrorReturn}} = b.breaker.Do(func() error {
		{{range $i, $r := .Results}}{{if $i}},{{end}}{{.Name}}{{end}} = b.source.{{.Name}}({{range .Params}}{{if eq .Ellipsis true}}{{.Name}}...{{else}}{{.Name}},{{end}}{{end}})
		return {{$ErrorReturn}}
	})
	return {{range $i, $r :=.Results}}{{if $i}},{{end}}{{.Name}}{{end}}
	{{- end}}
}

{{end -}}
{{- end}}
`
