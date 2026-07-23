package parse

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

const sampleClassifierHTML = `
<html><body>
<ul class="sidebar-section__classifiers">
  <li><strong>Programming Language</strong>
    <ul>
      <li><a href="/search/?c=Programming+Language+%3A%3A+Python+%3A%3A+3">Python :: 3</a></li>
    </ul>
  </li>
  <li><strong>License</strong>
    <ul>
      <li><a href="/search/?c=License+%3A%3A+OSI+Approved+%3A%3A+MIT">OSI Approved :: MIT</a></li>
    </ul>
  </li>
</ul>
</body></html>
`

func TestParseClassifiers(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(sampleClassifierHTML))
	if err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}

	classifiers, tagged := parseClassifiers(doc, "samplepkg", "1.0.0")

	if len(classifiers) != 2 {
		t.Fatalf("expected 2 classifiers, got %d", len(classifiers))
	}
	if classifiers[0].Category != "Programming Language" || classifiers[0].Value != "Python :: 3" {
		t.Fatalf("unexpected classifier 0: %+v", classifiers[0])
	}
	if classifiers[1].Category != "License" || classifiers[1].Value != "OSI Approved :: MIT" {
		t.Fatalf("unexpected classifier 1: %+v", classifiers[1])
	}

	if len(tagged) != 2 {
		t.Fatalf("expected 2 tagged_with rows, got %d", len(tagged))
	}
	if tagged[0].PackageName != "samplepkg" || tagged[0].Version != "1.0.0" || tagged[0].Category != "Programming Language" {
		t.Fatalf("unexpected tagged_with 0: %+v", tagged[0])
	}
}

// Fixture ini mereproduksi duplikasi sidebar (tab visible + tersembunyi) untuk cegah regresi penggandaan classifiers/tagged_with.
const duplicatedClassifierHTML = `
<html><body>
<div class="vertical-tabs__tabs">
<ul class="sidebar-section__classifiers">
  <li><strong>Programming Language</strong>
    <ul>
      <li><a href="/search/?c=Programming+Language+%3A%3A+Python+%3A%3A+3">Python :: 3</a></li>
    </ul>
  </li>
</ul>
</div>
<div class="vertical-tabs__content" style="display:none">
<ul class="sidebar-section__classifiers">
  <li><strong>Programming Language</strong>
    <ul>
      <li><a href="/search/?c=Programming+Language+%3A%3A+Python+%3A%3A+3">Python :: 3</a></li>
    </ul>
  </li>
</ul>
</div>
</body></html>
`

func TestParseClassifiersIgnoresDuplicateHiddenSidebar(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(duplicatedClassifierHTML))
	if err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}

	classifiers, tagged := parseClassifiers(doc, "samplepkg", "1.0.0")

	if len(classifiers) != 1 {
		t.Fatalf("expected 1 classifier (not doubled to 2), got %d: %+v", len(classifiers), classifiers)
	}
	if len(tagged) != 1 {
		t.Fatalf("expected 1 tagged_with row (not doubled to 2), got %d: %+v", len(tagged), tagged)
	}
}
