package writer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"pxs/internal/parse"
)

var entityFilenames = []string{
	"package.json", "organization.json", "maintainer.json", "maintained_by.json",
	"release.json", "release_detail.json", "release_file.json", "file_hash.json",
	"project_link.json", "classifier.json", "tagged_with.json", "release_keyword.json",
	"release_extra.json", "release_file_tag.json", "attestation.json",
}

// Writer: stream entity langsung ke disk (temp dir) supaya peak memory konstan.
type Writer struct {
	dir              string
	files            map[string]*os.File
	firstRow         map[string]bool
	seenOrgs         map[string]bool
	seenMaintainer   map[string]bool
	seenClassifier   map[string]bool
	seenPackage      map[string]bool
	seenMaintainedBy map[string]bool
}

func New() (*Writer, error) {
	dir, err := os.MkdirTemp("", "pxs-*")
	if err != nil {
		return nil, fmt.Errorf("membuat temp dir: %w", err)
	}

	w := &Writer{
		dir:              dir,
		files:            make(map[string]*os.File),
		firstRow:         make(map[string]bool),
		seenOrgs:         make(map[string]bool),
		seenMaintainer:   make(map[string]bool),
		seenClassifier:   make(map[string]bool),
		seenPackage:      make(map[string]bool),
		seenMaintainedBy: make(map[string]bool),
	}

	for _, name := range entityFilenames {
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("membuat file %s: %w", name, err)
		}
		if _, err := f.WriteString("["); err != nil {
			return nil, fmt.Errorf("menulis header %s: %w", name, err)
		}
		w.files[name] = f
		w.firstRow[name] = true
	}

	return w, nil
}

func (w *Writer) appendRow(filename string, row any) error {
	f := w.files[filename]
	data, err := json.Marshal(row)
	if err != nil {
		return fmt.Errorf("encode row untuk %s: %w", filename, err)
	}
	if !w.firstRow[filename] {
		if _, err := f.WriteString(","); err != nil {
			return err
		}
	}
	w.firstRow[filename] = false
	_, err = f.Write(data)
	return err
}

func (w *Writer) AddPackage(p parse.Package) error {
	if w.seenPackage[p.Name] {
		return nil
	}
	w.seenPackage[p.Name] = true
	return w.appendRow("package.json", p)
}

func (w *Writer) AddOrganization(o parse.Organization) error {
	if w.seenOrgs[o.Name] {
		return nil
	}
	w.seenOrgs[o.Name] = true
	return w.appendRow("organization.json", o)
}

// HasMaintainer: maintainer yang sama muncul di tiap versi historis, jadi harus dicek supaya profil gagal tidak dicoba ulang tiap halaman.
func (w *Writer) HasMaintainer(username string) bool {
	return w.seenMaintainer[username]
}

// MarkMaintainerAttempted: catat percobaan gagal tanpa menulis baris maintainer.json, agar tidak jadi joined_at kosong palsu.
func (w *Writer) MarkMaintainerAttempted(username string) {
	w.seenMaintainer[username] = true
}

func (w *Writer) AddMaintainer(m parse.Maintainer) error {
	if w.seenMaintainer[m.Username] {
		return nil
	}
	w.seenMaintainer[m.Username] = true
	return w.appendRow("maintainer.json", m)
}

func (w *Writer) AddMaintainedBy(m parse.MaintainedBy) error {
	key := m.PackageName + "\x00" + m.MaintainerUsername
	if w.seenMaintainedBy[key] {
		return nil
	}
	w.seenMaintainedBy[key] = true
	return w.appendRow("maintained_by.json", m)
}

func (w *Writer) AddRelease(r parse.Release) error { return w.appendRow("release.json", r) }

func (w *Writer) AddReleaseDetail(d parse.ReleaseDetail) error {
	return w.appendRow("release_detail.json", d)
}

func (w *Writer) AddReleaseFile(f parse.ReleaseFile) error {
	return w.appendRow("release_file.json", f)
}

func (w *Writer) AddFileHash(h parse.FileHash) error { return w.appendRow("file_hash.json", h) }

func (w *Writer) AddProjectLink(l parse.ProjectLink) error {
	return w.appendRow("project_link.json", l)
}

func (w *Writer) AddClassifier(c parse.Classifier) error {
	key := c.Category + "\x00" + c.Value
	if w.seenClassifier[key] {
		return nil
	}
	w.seenClassifier[key] = true
	return w.appendRow("classifier.json", c)
}

func (w *Writer) AddTaggedWith(t parse.TaggedWith) error { return w.appendRow("tagged_with.json", t) }

func (w *Writer) AddReleaseKeyword(packageName, version, keyword string) error {
	return w.appendRow("release_keyword.json", map[string]string{
		"package_name": packageName, "version": version, "keyword": keyword,
	})
}

func (w *Writer) AddReleaseExtra(packageName, version, extraName string) error {
	return w.appendRow("release_extra.json", map[string]string{
		"package_name": packageName, "version": version, "extra_name": extraName,
	})
}

func (w *Writer) AddReleaseFileTag(t parse.ReleaseFileTag) error {
	return w.appendRow("release_file_tag.json", t)
}

func (w *Writer) AddAttestation(a parse.Attestation) error {
	return w.appendRow("attestation.json", a)
}

// Finalize: aman dipanggil setelah run parsial — baris yang sudah ter-stream tetap jadi JSON valid.
func (w *Writer) Finalize(outputDir string) error {
	for _, name := range entityFilenames {
		f := w.files[name]
		if _, err := f.WriteString("]"); err != nil {
			return fmt.Errorf("menulis penutup %s: %w", name, err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("menutup file %s: %w", name, err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(outputDir), 0o755); err != nil {
		return fmt.Errorf("membuat parent output dir: %w", err)
	}
	if err := os.Rename(w.dir, outputDir); err != nil {
		return fmt.Errorf("memindahkan temp dir ke output: %w", err)
	}
	return nil
}
