package parse

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

const samplePackageHeaderHTML = `
<html><body>
<div class="package-header__right">
  <span class="status-badge"><span>This project has been quarantined</span></span>
</div>
<div class="sidebar-section verified">
  <h6>Owner</h6>
  <ul class="vertical-tabs__list">
    <li><a href="/org/exampleorg/">Example Org</a></li>
  </ul>
</div>
</body></html>
`

func TestParsePackageAndOrg(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(samplePackageHeaderHTML))
	if err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}

	pkg, org := parsePackageAndOrg(doc)

	if pkg.LifecycleStatus != "archived" && pkg.LifecycleStatus != "deprecated" && pkg.LifecycleStatus != "quarantined" {
		t.Fatalf("expected lifecycle_status to be quarantined, got %q", pkg.LifecycleStatus)
	}
	if pkg.OrganizationOwner != "exampleorg" {
		t.Fatalf("expected organization_owner 'exampleorg', got %q", pkg.OrganizationOwner)
	}
	if org == nil {
		t.Fatal("expected organization to be non-nil")
	}
	if org.Name != "exampleorg" || org.DisplayName != "Example Org" {
		t.Fatalf("unexpected organization: %+v", org)
	}
}

func TestParsePackageAndOrg_NoOwner(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<html><body></body></html>`))
	if err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}

	pkg, org := parsePackageAndOrg(doc)
	if org != nil {
		t.Fatalf("expected nil organization, got %+v", org)
	}
	if pkg.OrganizationOwner != "" {
		t.Fatalf("expected empty organization_owner, got %q", pkg.OrganizationOwner)
	}
}
