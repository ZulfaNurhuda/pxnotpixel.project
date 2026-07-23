package parse

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

const sampleHistoryHTML = `
<html><body>
<div class="release-timeline">
  <div class="release">
    <div class="release__version">2.0.0</div>
  </div>
  <div class="release">
    <div class="release__version">1.9.0 <span class="badge--warning">pre-release</span></div>
  </div>
  <div class="release">
    <div class="release__version">1.8.0 <span class="badge--danger">yanked</span></div>
    <div class="release__yanked-reason">Broken build</div>
  </div>
</div>
</body></html>
`

func TestParseReleaseHistory(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(sampleHistoryHTML))
	if err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}

	entries := parseReleaseHistory(doc)

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Version != "2.0.0" || entries[0].IsPrerelease || entries[0].Yanked {
		t.Fatalf("unexpected entry 0: %+v", entries[0])
	}
	if entries[1].Version != "1.9.0" || !entries[1].IsPrerelease || entries[1].Yanked {
		t.Fatalf("unexpected entry 1: %+v", entries[1])
	}
	if entries[2].Version != "1.8.0" || !entries[2].Yanked || entries[2].YankedReason != "Broken build" {
		t.Fatalf("unexpected entry 2: %+v", entries[2])
	}
}
