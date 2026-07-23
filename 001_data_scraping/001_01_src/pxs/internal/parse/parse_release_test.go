package parse

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

// sampleReleaseHTML: label diawali newline/indentasi, li License Expression punya footer SPDX setelah value, span keyword punya newline SEBELUM koma.
const sampleReleaseHTML = `
<html><body>
<div class="package-header__date">
Released: <time datetime="2026-01-02T10:00:00+0000">Jan 2, 2026</time>        </div>
<p class="package-description__summary">A sample package.</p>
<div class="project-description"><p>Full <b>description</b>.</p></div>
<div class="sidebar-section verified">
  <h6>Meta</h6>
  <li>
            <span>
              <strong>Author:</strong> <a href="mailto:alice@example.com">Alice Example</a>
            </span>
          </li>
</div>
<div class="sidebar-section unverified">
  <h6>Meta</h6>
  <li>
            <span>
              <strong>License Expression:</strong> MIT
              <br>
              <small>
                <i>
                  <a href="https://spdx.org/licenses/">SPDX</a>
                  <a href="https://spdx.github.io/spdx-spec/">License Expression</a>
                </i>
              </small>
            </span>
          </li>
  <li>
            <span>
              <strong>Requires:</strong> Python &gt;=3.9
            </span>
          </li>
  <li>
            <span>
              <strong>Maintainer:</strong> <a href="mailto:bob@example.com">Bob Example</a>
            </span>
          </li>
  <li class="tags"><span class="package-keyword">
                  alpha
,                </span><span class="package-keyword">
                  beta
,                </span></li>
  <li><strong>Provides-Extra:</strong> <code>test</code> <code>docs</code></li>
</div>
</body></html>
`

func TestParseReleaseMeta(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(sampleReleaseHTML))
	if err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}

	release, detail := parseReleaseMeta(doc, "samplepkg", "1.0.0")

	if release.PackageName != "samplepkg" || release.Version != "1.0.0" {
		t.Fatalf("unexpected identity: %+v", release)
	}
	if release.Created != "2026-01-02T10:00:00+0000" {
		t.Fatalf("unexpected created: %q (should read the <time> datetime attribute, not display text)", release.Created)
	}
	if release.Summary != "A sample package." {
		t.Fatalf("unexpected summary: %q", release.Summary)
	}
	if release.License != "MIT" {
		t.Fatalf("unexpected license: %q (SPDX footer text must not leak in)", release.License)
	}
	if release.RequiresPython != ">=3.9" {
		t.Fatalf("unexpected requires_python: %q", release.RequiresPython)
	}
	if len(release.Keywords) != 2 || release.Keywords[0] != "alpha" || release.Keywords[1] != "beta" {
		t.Fatalf("unexpected keywords: %v", release.Keywords)
	}
	if len(release.ProvidesExtra) != 2 || release.ProvidesExtra[0] != "test" || release.ProvidesExtra[1] != "docs" {
		t.Fatalf("unexpected provides_extra: %v", release.ProvidesExtra)
	}

	if detail.PackageName != "samplepkg" || detail.Version != "1.0.0" {
		t.Fatalf("unexpected detail identity: %+v", detail)
	}
	if detail.Description != "<p>Full <b>description</b>.</p>" {
		t.Fatalf("unexpected description: %q", detail.Description)
	}
	if detail.MetaAuthor != "Alice Example" {
		t.Fatalf("unexpected meta_author: %q (label prefix must be stripped)", detail.MetaAuthor)
	}
	if detail.MetaAuthorEmail != "alice@example.com" || !detail.MetaAuthorEmailVerified {
		t.Fatalf("unexpected author: %+v", detail)
	}
	if detail.MetaMaintainer != "Bob Example" {
		t.Fatalf("unexpected meta_maintainer: %q (label prefix must be stripped)", detail.MetaMaintainer)
	}
	if detail.MetaMaintainerEmail != "bob@example.com" || detail.MetaMaintainerEmailVerified {
		t.Fatalf("unexpected maintainer: %+v", detail)
	}
}

// Fixture ini mereproduksi duplikasi sidebar untuk cegah regresi penggandaan keyword.
const duplicatedKeywordsHTML = `
<html><body>
<div class="package-header__date"><time datetime="2026-01-02T10:00:00+0000"></time></div>
<div class="vertical-tabs__tabs">
<div class="sidebar-section unverified">
  <h6>Meta</h6>
  <li class="tags"><span class="package-keyword">alpha,</span></li>
</div>
</div>
<div class="vertical-tabs__content" style="display:none">
<div class="sidebar-section unverified">
  <h6>Meta</h6>
  <li class="tags"><span class="package-keyword">alpha,</span></li>
</div>
</div>
</body></html>
`

func TestParseReleaseMetaIgnoresDuplicateHiddenSidebarKeywords(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(duplicatedKeywordsHTML))
	if err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}

	release, _ := parseReleaseMeta(doc, "samplepkg", "1.0.0")

	if len(release.Keywords) != 1 || release.Keywords[0] != "alpha" {
		t.Fatalf("expected 1 keyword (not doubled to 2), got %v", release.Keywords)
	}
}
