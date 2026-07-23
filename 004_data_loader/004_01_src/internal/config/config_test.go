package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadDBPassword(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	content := "POSTGRES_USER=px_user\nPOSTGRES_PASSWORD=s3cret!\nPOSTGRES_DB=px_db\n"
	if err := os.WriteFile(envPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ReadDBPassword(envPath)
	if err != nil {
		t.Fatalf("ReadDBPassword returned error: %v", err)
	}
	if got != "s3cret!" {
		t.Fatalf("got %q, want %q", got, "s3cret!")
	}
}

func TestReadDBPasswordMissingKey(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("POSTGRES_USER=px_user\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := ReadDBPassword(envPath); err == nil {
		t.Fatal("expected error for missing POSTGRES_PASSWORD, got nil")
	}
}

func TestReadDBPasswordMissingFile(t *testing.T) {
	if _, err := ReadDBPassword(filepath.Join(t.TempDir(), "does-not-exist.env")); err == nil {
		t.Fatal("expected error for missing .env file, got nil")
	}
}

func TestFindLatestRun(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"pxs_20260101T000000", "pxs_20260723T044432", "pxs_20260215T101010"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Entri non-pxs_ harus diabaikan.
	if err := os.Mkdir(filepath.Join(dir, "not_a_run"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := FindLatestRun(dir)
	if err != nil {
		t.Fatalf("FindLatestRun returned error: %v", err)
	}
	want := filepath.Join(dir, "pxs_20260723T044432")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFindLatestRunNoneFound(t *testing.T) {
	if _, err := FindLatestRun(t.TempDir()); err == nil {
		t.Fatal("expected error when no pxs_* folders exist, got nil")
	}
}
