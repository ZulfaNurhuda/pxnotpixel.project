package load

import "testing"

func TestLookupPackageFound(t *testing.T) {
	r := NewResolver()
	r.packageID["requests"] = "id-123"

	got, err := r.lookupPackage("requests")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "id-123" {
		t.Fatalf("got %q, want %q", got, "id-123")
	}
}

func TestLookupPackageMissing(t *testing.T) {
	r := NewResolver()

	_, err := r.lookupPackage("does-not-exist")
	if err == nil {
		t.Fatal("expected error for unresolved package reference, got nil")
	}
}

func TestLookupReleaseCompositeKey(t *testing.T) {
	r := NewResolver()
	r.releaseID[releaseKey{PackageName: "requests", Version: "2.31.0"}] = "rel-1"

	got, err := r.lookupRelease("requests", "2.31.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "rel-1" {
		t.Fatalf("got %q, want %q", got, "rel-1")
	}

	if _, err := r.lookupRelease("requests", "9.9.9"); err == nil {
		t.Fatal("expected error for unresolved release reference, got nil")
	}
}

func TestLookupFileCompositeKey(t *testing.T) {
	r := NewResolver()
	key := fileKey{PackageName: "requests", Version: "2.31.0", Filename: "requests-2.31.0.whl"}
	r.fileID[key] = "file-1"

	got, err := r.lookupFile("requests", "2.31.0", "requests-2.31.0.whl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "file-1" {
		t.Fatalf("got %q, want %q", got, "file-1")
	}

	if _, err := r.lookupFile("requests", "2.31.0", "wrong-file.whl"); err == nil {
		t.Fatal("expected error for unresolved file reference, got nil")
	}
}

func TestLookupClassifierCompositeKey(t *testing.T) {
	r := NewResolver()
	r.classifierID[classifierKey{Category: "License", Value: "OSI Approved :: MIT License"}] = "cls-1"

	got, err := r.lookupClassifier("License", "OSI Approved :: MIT License")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "cls-1" {
		t.Fatalf("got %q, want %q", got, "cls-1")
	}

	if _, err := r.lookupClassifier("License", "does-not-exist"); err == nil {
		t.Fatal("expected error for unresolved classifier reference, got nil")
	}
}

func TestLookupMaintainerMissing(t *testing.T) {
	r := NewResolver()
	if _, err := r.lookupMaintainer("nobody"); err == nil {
		t.Fatal("expected error for unresolved maintainer reference, got nil")
	}
}

func TestLookupOrgEmptyNameIsNilNotError(t *testing.T) {
	r := NewResolver()

	got, err := r.lookupOrg("")
	if err != nil {
		t.Fatalf("empty org name should not error, got: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil org id for empty name, got %v", *got)
	}
}

func TestLookupOrgFoundAndMissing(t *testing.T) {
	r := NewResolver()
	r.orgID["Python Software Foundation"] = "org-1"

	got, err := r.lookupOrg("Python Software Foundation")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || *got != "org-1" {
		t.Fatalf("got %v, want org-1", got)
	}

	if _, err := r.lookupOrg("Unknown Org"); err == nil {
		t.Fatal("expected error for unresolved org reference, got nil")
	}
}
