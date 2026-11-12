package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoveryAndFreshness(t *testing.T) {
	t.Parallel()
	executable := filepath.Join(t.TempDir(), "genmigrations")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", executable, ".") //nolint:gosec // G204: executes Go or the generator binary built into this test's private temporary directory.
	output, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("build generator: %v: %s", err, output)
	}
	root := t.TempDir()
	write(t, filepath.Join(root, "go.mod"), "module example.test/discovery\n")
	write(t, filepath.Join(root, "internal", "mindflow", "storage", "placeholder.go"), "package storage\n")
	addDomain(t, root, "canvas")
	run := func(check bool) error {
		arguments := []string{}
		if check {
			arguments = append(arguments, "-check")
		}
		command := exec.CommandContext(t.Context(), executable, arguments...) //nolint:gosec // G204: executes Go or the generator binary built into this test's private temporary directory.
		command.Dir = root

		return command.Run()
	}
	err = run(false)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "internal", "mindflow", "storage", "migrations_generated.go")
	before, err := os.ReadFile(target) //nolint:gosec // G304: target is the generated catalog in this test's private temporary directory.
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(before), "canvasmigrations.Files()") {
		t.Fatal("canvas migrations were not discovered")
	}
	err = run(true)
	if err != nil {
		t.Fatal(err)
	}
	addDomain(t, root, "invitation")
	if run(true) == nil {
		t.Fatal("new migration package did not make the catalog stale")
	}
	err = run(false)
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(target) //nolint:gosec // G304: target is the generated catalog in this test's private temporary directory.
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "invitationmigrations.Files()") {
		t.Fatal("new domain was not linked")
	}
	err = run(false)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := os.ReadFile(target) //nolint:gosec // G304: target is the generated catalog in this test's private temporary directory.
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, repeated) {
		t.Fatal("migration discovery is nondeterministic")
	}
}

func addDomain(t *testing.T, root, name string) {
	t.Helper()
	write(t, filepath.Join(root, "internal", name, "storage", "migrations", "migrations.go"), "package migrations\n")
}

func write(t *testing.T, path, body string) {
	t.Helper()
	err := os.MkdirAll(filepath.Dir(path), 0o700)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(path, []byte(body), 0o600)
	if err != nil {
		t.Fatal(err)
	}
}
