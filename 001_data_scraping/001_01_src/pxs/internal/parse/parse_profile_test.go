package parse

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

// sampleProfileHTML: label aksesibilitas "Date joined" mendahului teks visible "Joined"; tanggal asli ada di elemen <time>.
const sampleProfileHTML = `
<html><body>
<div class="author-profile__metadiv">
  <i class="fa fa-user-circle" aria-hidden="true"></i>
  <span class="sr-only">Username</span>
  <span class="break">somebody</span>
</div>
<div class="author-profile__metadiv">
  <i class="fa fa-calendar-alt" aria-hidden="true"></i>
  <span class="sr-only">Date joined</span>
  Joined <time datetime="2020-01-02T00:00:00+0000">Jan 2, 2020</time>
</div>
</body></html>
`

func TestParseMaintainerJoinedAt(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(sampleProfileHTML))
	if err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}

	joinedAt := parseMaintainerJoinedAt(doc)
	if joinedAt != "2020-01-02T00:00:00+0000" {
		t.Fatalf("unexpected joined_at: %q (should read the <time> datetime attribute, not display text)", joinedAt)
	}
}

// Sebagian akun tidak render metadiv "Date joined" sama sekali — harus jadi "", bukan error.
func TestParseMaintainerJoinedAt_NoDateJoinedShown(t *testing.T) {
	const html = `
<html><body>
<div class="author-profile__metadiv">
  <i class="fa fa-user-circle" aria-hidden="true"></i>
  <span class="sr-only">Username</span>
  <span class="break">aws</span>
</div>
</body></html>
`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}

	joinedAt := parseMaintainerJoinedAt(doc)
	if joinedAt != "" {
		t.Fatalf("expected empty joined_at when no Date joined metadiv exists, got %q", joinedAt)
	}
}
