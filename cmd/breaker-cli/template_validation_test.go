package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTemplateValidationPipeline(t *testing.T) {
	workdir := prepareValidationFixture(t)
	output := filepath.Join(workdir, "wrapper", "filterbreaker.go")

	pkg := Package{
		Name:       "sourcepkg",
		ImportPath: "example.com/templatevalidate/sourcepkg",
		Structs: []Struct{
			{
				Name: "FilterBreaker",
				Implemented: Interface{
					Name: "Filter",
					Methods: []Method{
						{
							Name:   "Filter",
							Params: []Param{{Name: "elements", Type: "[]string"}},
							Results: []Param{
								{Name: "filtered", Type: "[]string"},
								{Name: "err", Type: "error"},
							},
						},
					},
				},
			},
		},
	}

	t.Run("parse", func(t *testing.T) {
		if _, err := parseBreakerTemplate(); err != nil {
			t.Fatalf("parse template: %v", err)
		}
	})

	t.Run("generate_and_go_fix", func(t *testing.T) {
		reader, err := generate(context.Background(), pkg, output)
		if err != nil {
			t.Fatalf("generate: %v", err)
		}

		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("read generated code: %v", err)
		}

		if err = os.WriteFile(output, data, 0o644); err != nil {
			t.Fatalf("write generated code: %v", err)
		}

		diff := runCommandOutput(t, filepath.Join(workdir, "wrapper"), "go", "fix", "-diff", ".")
		if strings.TrimSpace(diff) != "" {
			t.Fatalf("go fix -diff produced changes; update cmd/breaker-cli/breaker.gohtml or generation logic:\n%s", diff)
		}
	})

	t.Run("final_build", func(t *testing.T) {
		runCommand(t, filepath.Join(workdir, "wrapper"), "go", "build", ".")
	})
}

func prepareValidationFixture(t *testing.T) string {
	t.Helper()

	workdir := t.TempDir()
	repoRoot, err := filepath.Abs(filepath.Join(".", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	modFile := fmt.Sprintf(`module example.com/templatevalidate

go 1.26

require github.com/chenyanchen/breaker v0.0.0

replace github.com/chenyanchen/breaker => %s
`, filepath.ToSlash(repoRoot))

	if err = os.WriteFile(filepath.Join(workdir, "go.mod"), []byte(modFile), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	if err = os.MkdirAll(filepath.Join(workdir, "sourcepkg"), 0o755); err != nil {
		t.Fatalf("create sourcepkg dir: %v", err)
	}
	if err = os.MkdirAll(filepath.Join(workdir, "wrapper"), 0o755); err != nil {
		t.Fatalf("create wrapper dir: %v", err)
	}

	source := `package sourcepkg

type Filter interface {
	Filter(elements []string) ([]string, error)
}
`
	if err = os.WriteFile(filepath.Join(workdir, "sourcepkg", "source.go"), []byte(source), 0o644); err != nil {
		t.Fatalf("write source package: %v", err)
	}

	return workdir
}

func runCommand(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	_ = runCommandOutput(t, dir, name, args...)
}

func runCommandOutput(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, output)
	}
	return string(output)
}
