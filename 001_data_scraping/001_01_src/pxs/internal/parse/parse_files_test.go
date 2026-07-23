package parse

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

// sampleFilesHTML: #files cuma berisi summary list (untuk klasifikasi sdist/wheel), detail panel tiap file ada di div[id] terpisah, bukan nested.
const sampleFilesHTML = `
<html><body>
<div id="files">
  <h2 class="page-title">Download files</h2>
  <p>Download the file for your platform. <a href="https://packaging.python.org/tutorials/installing-packages/">installing packages</a>.</p>
  <h3>Source Distribution</h3>
  <div class="file">
    <div class="card file__card">
      <a href="https://files.pythonhosted.org/packages/src/samplepkg-1.0.0.tar.gz">samplepkg-1.0.0.tar.gz</a>
      (1.2 kB <a href="#samplepkg-1.0.0.tar.gz">view details</a>)
    </div>
  </div>
  <h3>Built Distribution</h3>
  <div class="file">
    <div class="card file__card">
      <a href="https://files.pythonhosted.org/packages/whl/samplepkg-1.0.0-py3-none-any.whl">samplepkg-1.0.0-py3-none-any.whl</a>
      (5.5 kB <a href="#samplepkg-1.0.0-py3-none-any.whl">view details</a>)
    </div>
  </div>
</div>
<div id="samplepkg-1.0.0.tar.gz">
  <h2 class="page-title">File details</h2>
  <p>Details for the file <code>samplepkg-1.0.0.tar.gz</code>.</p>
  <h3>File metadata</h3>
  <ul>
    <li>Download URL: <a href="/packages/src/samplepkg-1.0.0.tar.gz">samplepkg-1.0.0.tar.gz</a></li>
    <li>Upload date: <time datetime="2026-01-02T00:00:00+0000">Jan 2, 2026</time></li>
    <li>Size: 1.2 kB</li>
    <li>Tags: Source</li>
    <li>Uploaded using Trusted Publishing? Yes</li>
    <li>Uploaded via: twine/5.0</li>
  </ul>
  <h3>File hashes</h3>
  <table class="table table--hashes">
    <tbody>
      <tr><th scope="row">SHA256</th><td><code>abc123</code></td></tr>
      <tr><th scope="row">MD5</th><td><code>def456</code></td></tr>
      <tr><th scope="row">BLAKE2b-256</th><td><code>ghi789</code></td></tr>
    </tbody>
  </table>
</div>
<div id="samplepkg-1.0.0-py3-none-any.whl">
  <h2 class="page-title">File details</h2>
  <p>Details for the file <code>samplepkg-1.0.0-py3-none-any.whl</code>.</p>
  <h3>File metadata</h3>
  <ul>
    <li>Download URL: <a href="/packages/whl/samplepkg-1.0.0-py3-none-any.whl">samplepkg-1.0.0-py3-none-any.whl</a></li>
    <li>Upload date: <time datetime="2026-01-02T00:00:00+0000">Jan 2, 2026</time></li>
    <li>Size: 5.5 kB</li>
    <li>Tags: Python 3</li>
    <li>Uploaded using Trusted Publishing? No</li>
  </ul>
  <h3>File hashes</h3>
  <table class="table table--hashes">
    <tbody>
      <tr><th scope="row">SHA256</th><td><code>jkl012</code></td></tr>
      <tr><th scope="row">MD5</th><td><code>mno345</code></td></tr>
      <tr><th scope="row">BLAKE2b-256</th><td><code>pqr678</code></td></tr>
    </tbody>
  </table>
</div>
</body></html>
`

func TestParseReleaseFiles(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(sampleFilesHTML))
	if err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}

	files, hashes, tags, _ := parseReleaseFiles(doc, "samplepkg", "1.0.0")

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0].Filename != "samplepkg-1.0.0.tar.gz" || files[0].PackageType != "sdist" {
		t.Fatalf("unexpected sdist file: %+v", files[0])
	}
	// 1.2 kB * 1024 = 1228.8, dibulatkan ke bawah jadi 1228 bytes.
	if files[0].Size != 1228 || !files[0].IsTrustedPublishing || files[0].UploadedVia != "twine/5.0" {
		t.Fatalf("unexpected sdist metadata: %+v", files[0])
	}
	if files[0].UploadTime != "2026-01-02T00:00:00+0000" {
		t.Fatalf("unexpected upload_time: %q (should read the <time> datetime attribute)", files[0].UploadTime)
	}
	if files[1].Filename != "samplepkg-1.0.0-py3-none-any.whl" || files[1].PackageType != "wheel" {
		t.Fatalf("unexpected wheel file: %+v", files[1])
	}
	if files[1].IsTrustedPublishing {
		t.Fatalf("expected wheel IsTrustedPublishing=false, got true")
	}

	if len(hashes) != 6 {
		t.Fatalf("expected 6 hash rows (3 per file), got %d", len(hashes))
	}
	if hashes[0].Algorithm != "SHA256" || hashes[0].Digest != "abc123" || hashes[0].Filename != "samplepkg-1.0.0.tar.gz" {
		t.Fatalf("unexpected first hash: %+v", hashes[0])
	}

	// Cuma wheel yang dapat baris tag ("Tags: Source" pada sdist dilewati).
	if len(tags) != 1 {
		t.Fatalf("expected 1 wheel tag row, got %d: %+v", len(tags), tags)
	}
	if tags[0].WheelTag != "Python 3" || tags[0].Filename != "samplepkg-1.0.0-py3-none-any.whl" {
		t.Fatalf("unexpected tag: %+v", tags[0])
	}
}

func TestParseFileSize(t *testing.T) {
	cases := []struct {
		text string
		want int64
	}{
		{"1.2 kB", 1228},
		{"512 Bytes", 512},
		{"228.1 kB", 233574},
		{"1.5 MB", 1572864},
		{"", 0},
	}
	for _, c := range cases {
		got := parseFileSize(c.text)
		if got != c.want {
			t.Errorf("parseFileSize(%q) = %d, want %d", c.text, got, c.want)
		}
	}
}
