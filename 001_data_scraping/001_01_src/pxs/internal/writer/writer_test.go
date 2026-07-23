package writer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"pxs/internal/parse"
)

func TestWriter_StreamsAndDedupsAndFinalizes(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	w.AddPackage(parse.Package{Name: "samplepkg"})
	w.AddOrganization(parse.Organization{Name: "org1", DisplayName: "Org One"})
	w.AddOrganization(parse.Organization{Name: "org1", DisplayName: "Org One"}) // duplikat, harus dedup
	w.AddMaintainer(parse.Maintainer{Username: "alice", JoinedAt: "Jan 2, 2020"})
	w.AddMaintainer(parse.Maintainer{Username: "alice", JoinedAt: "Jan 2, 2020"}) // duplikat
	w.AddClassifier(parse.Classifier{Category: "License", Value: "MIT"})
	w.AddClassifier(parse.Classifier{Category: "License", Value: "MIT"}) // duplikat
	w.AddRelease(parse.Release{PackageName: "samplepkg", Version: "1.0.0"})

	outputDir := t.TempDir()
	finalDir := filepath.Join(outputDir, "pxs_20260722120000")
	if err := w.Finalize(finalDir); err != nil {
		t.Fatalf("Finalize() error: %v", err)
	}

	assertJSONArrayLen := func(filename string, want int) {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(finalDir, filename))
		if err != nil {
			t.Fatalf("reading %s: %v", filename, err)
		}
		var rows []json.RawMessage
		if err := json.Unmarshal(data, &rows); err != nil {
			t.Fatalf("%s is not valid JSON array: %v (content: %s)", filename, err, data)
		}
		if len(rows) != want {
			t.Fatalf("%s: expected %d rows, got %d", filename, want, len(rows))
		}
	}

	assertJSONArrayLen("package.json", 1)
	assertJSONArrayLen("organization.json", 1) // dedup 2 -> 1
	assertJSONArrayLen("maintainer.json", 1)   // dedup 2 -> 1
	assertJSONArrayLen("classifier.json", 1)   // dedup 2 -> 1
	assertJSONArrayLen("release.json", 1)
}

func TestWriter_IsAlreadyWritten_UsedForMaintainerFetchSkip(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if w.HasMaintainer("alice") {
		t.Fatal("expected HasMaintainer to be false before adding")
	}
	w.AddMaintainer(parse.Maintainer{Username: "alice", JoinedAt: "Jan 2, 2020"})
	if !w.HasMaintainer("alice") {
		t.Fatal("expected HasMaintainer to be true after adding")
	}

	if err := w.Finalize(filepath.Join(t.TempDir(), "out")); err != nil {
		t.Fatalf("Finalize() error: %v", err)
	}
}
