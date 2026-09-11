package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestListFiles(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"pkg", filepath.Join("pkg", "nested"), "vendor"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"main.go", filepath.Join("pkg", "p.go"), filepath.Join("pkg", "nested", "n.txt"), filepath.Join("vendor", "ignored.go")} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("package p\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got := ListFiles(root, func(path string) bool { return strings.HasSuffix(path, ".go") })
	for i := range got {
		got[i], _ = filepath.Rel(root, got[i])
	}
	sort.Strings(got)
	want := []string{"main.go", filepath.Join("pkg", "p.go")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ListFiles() = %v, want %v", got, want)
	}
}

func TestIsExecutable(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "testbin")
	plain := filepath.Join(root, "plain.txt")
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}
	if err := os.WriteFile(executable, nil, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plain, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(executable, 0755); err != nil {
		t.Fatal(err)
	}
	if !IsExecutable(executable) {
		t.Fatalf("IsExecutable(%q) = false", executable)
	}
	if IsExecutable(plain) {
		t.Fatalf("IsExecutable(%q) = true", plain)
	}
}
