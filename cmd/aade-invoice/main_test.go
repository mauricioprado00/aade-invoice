package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveTemplatePathExplicit(t *testing.T) {
	const path = "somewhere/invoice.json"

	got, err := resolveTemplatePath(path, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Fatalf("resolveTemplatePath() = %q, want %q", got, path)
	}
}

func TestResolveTemplatePathDiscoversSingleJSONFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invoice.json")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "invoice-template.json.sample"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := resolveTemplatePath("", dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Fatalf("resolveTemplatePath() = %q, want %q", got, path)
	}
}

func TestResolveTemplatePathRequiresFlagWithoutExactlyOneJSONFile(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  string
	}{
		{name: "none", want: "found 0"},
		{name: "multiple", files: []string{"one.json", "two.json"}, want: "found 2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
					t.Fatal(err)
				}
			}

			_, err := resolveTemplatePath("", dir)
			if err == nil {
				t.Fatal("resolveTemplatePath() returned no error")
			}
			if !strings.Contains(err.Error(), "--template is required") || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("resolveTemplatePath() error = %q", err)
			}
		})
	}
}
