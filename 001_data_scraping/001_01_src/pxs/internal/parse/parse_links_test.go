package parse

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

const sampleLinksHTML = `
<html><body>
<div class="sidebar-section verified">
  <h6>Project links</h6>
  <ul class="vertical-tabs__list">
    <li><a href="https://example.com/homepage">Homepage</a></li>
  </ul>
</div>
<div class="sidebar-section unverified">
  <h6>Project links</h6>
  <ul class="vertical-tabs__list">
    <li><a href="https://example.com/docs">Documentation</a></li>
  </ul>
</div>
</body></html>
`

func TestParseProjectLinks(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(sampleLinksHTML))
	if err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}

	links := parseProjectLinks(doc, "samplepkg", "1.0.0")

	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
	if links[0].Label != "Homepage" || links[0].URL != "https://example.com/homepage" || !links[0].Verified {
		t.Fatalf("unexpected verified link: %+v", links[0])
	}
	if links[1].Label != "Documentation" || links[1].Verified {
		t.Fatalf("unexpected unverified link: %+v", links[1])
	}
}

// Fixture ini mereproduksi duplikasi sidebar (tab visible + tersembunyi) untuk cegah regresi penggandaan link.
const duplicatedSidebarLinksHTML = `
<html><body>
<div class="vertical-tabs__tabs">
  <div class="sidebar-section verified">
    <h6>Project links</h6>
    <ul class="vertical-tabs__list">
      <li><a href="https://example.com/homepage">Homepage</a></li>
    </ul>
  </div>
  <div class="sidebar-section unverified">
    <h6>Project links</h6>
    <ul class="vertical-tabs__list">
      <li><a href="https://example.com/docs">Documentation</a></li>
    </ul>
  </div>
</div>
<div class="vertical-tabs__content" style="display:none">
  <div class="sidebar-section verified">
    <h6>Project links</h6>
    <ul class="vertical-tabs__list">
      <li><a href="https://example.com/homepage">Homepage</a></li>
    </ul>
  </div>
  <div class="sidebar-section unverified">
    <h6>Project links</h6>
    <ul class="vertical-tabs__list">
      <li><a href="https://example.com/docs">Documentation</a></li>
    </ul>
  </div>
</div>
</body></html>
`

func TestParseProjectLinksIgnoresDuplicateHiddenSidebar(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(duplicatedSidebarLinksHTML))
	if err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}

	links := parseProjectLinks(doc, "samplepkg", "1.0.0")

	if len(links) != 2 {
		t.Fatalf("expected 2 links (not doubled to 4), got %d: %+v", len(links), links)
	}
}
