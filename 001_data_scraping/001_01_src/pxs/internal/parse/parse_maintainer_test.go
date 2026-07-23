package parse

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

const sampleMaintainersHTML = `
<html><body>
<div class="sidebar-section verified">
  <h6>Maintainers</h6>
  <span class="sidebar-section__user-gravatar-text">alice</span>
  <span class="sidebar-section__user-gravatar-text">bob</span>
</div>
</body></html>
`

func TestParseMaintainerUsernames(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(sampleMaintainersHTML))
	if err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}

	usernames, maintainedBy := parseMaintainerUsernames(doc, "samplepkg")

	if len(usernames) != 2 || usernames[0] != "alice" || usernames[1] != "bob" {
		t.Fatalf("unexpected usernames: %v", usernames)
	}
	if len(maintainedBy) != 2 {
		t.Fatalf("expected 2 maintained_by rows, got %d", len(maintainedBy))
	}
	if maintainedBy[0].PackageName != "samplepkg" || maintainedBy[0].MaintainerUsername != "alice" {
		t.Fatalf("unexpected maintained_by 0: %+v", maintainedBy[0])
	}
}
