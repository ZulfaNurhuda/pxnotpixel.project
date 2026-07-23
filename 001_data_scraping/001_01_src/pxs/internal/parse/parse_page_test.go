package parse

import "testing"

const sampleFullPageHTML = `
<html><body>
<div class="package-header__right"><span class="status-badge"><span>ok</span></span></div>
<div class="package-header__date">Released: Jan 2, 2026</div>
<p class="package-description__summary">A sample package.</p>
<div class="project-description"><p>Desc</p></div>
<div class="sidebar-section verified">
  <h6>Owner</h6>
  <ul class="vertical-tabs__list"><li><a href="/org/exampleorg/">Example Org</a></li></ul>
  <h6>Maintainers</h6>
  <span class="sidebar-section__user-gravatar-text">alice</span>
  <h6>Project links</h6>
  <ul class="vertical-tabs__list"><li><a href="https://example.com">Home</a></li></ul>
</div>
<div class="sidebar-section unverified">
  <h6>Meta</h6>
  <li><span><strong>License:</strong></span> MIT</li>
</div>
<ul class="sidebar-section__classifiers">
  <li><strong>License</strong><ul><li><a href="#">OSI Approved :: MIT</a></li></ul></li>
</ul>
<div class="release-timeline">
  <div class="release"><div class="release__version">1.0.0</div></div>
</div>
</body></html>
`

func TestParsePage(t *testing.T) {
	page, err := ParsePage([]byte(sampleFullPageHTML), "samplepkg", "1.0.0")
	if err != nil {
		t.Fatalf("ParsePage returned error: %v", err)
	}

	if page.Package.Name != "samplepkg" {
		t.Fatalf("unexpected package name: %q", page.Package.Name)
	}
	if page.Organization == nil || page.Organization.Name != "exampleorg" {
		t.Fatalf("unexpected organization: %+v", page.Organization)
	}
	if page.Release.License != "MIT" {
		t.Fatalf("unexpected license: %q", page.Release.License)
	}
	if len(page.Maintainers) != 1 || page.Maintainers[0] != "alice" {
		t.Fatalf("unexpected maintainers: %v", page.Maintainers)
	}
	if len(page.ProjectLinks) != 1 {
		t.Fatalf("expected 1 project link, got %d", len(page.ProjectLinks))
	}
	if len(page.Classifiers) != 1 {
		t.Fatalf("expected 1 classifier, got %d", len(page.Classifiers))
	}
	if len(page.ReleaseHistory) != 1 || page.ReleaseHistory[0].Version != "1.0.0" {
		t.Fatalf("unexpected release history: %+v", page.ReleaseHistory)
	}
}

func TestParsePage_InvalidHTML(t *testing.T) {
	_, err := ParsePage([]byte(""), "samplepkg", "1.0.0")
	if err != nil {
		t.Fatalf("empty (but well-formed) HTML should not error, got: %v", err)
	}
}
