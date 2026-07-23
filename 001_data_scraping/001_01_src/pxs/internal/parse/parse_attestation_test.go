package parse

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

const sampleAttestationHTML = `
<html><body>
<div id="samplepkg-1.0.0-py3-none-any.whl">
  <h4>Provenance</h4>
  <li>Sigstore transparency entry: <a href="https://search.sigstore.dev/?logIndex=123456">123456</a></li>
  <li>Sigstore integration time: <time datetime="2026-01-02T03:04:05+0000">Jan 2, 2026, 3:04:05 AM</time></li>
  <li>Statement type: <code>https://in-toto.io/Statement/v1</code></li>
  <li>Predicate type: <code>https://slsa.dev/provenance/v1</code></li>
  <li>Subject name: <code>samplepkg-1.0.0-py3-none-any.whl</code></li>
  <li>Subject digest: <code>abcdef123456</code></li>
  <div>
    <p>Source repository:</p>
    <li>Permalink: <code>github.com/example/samplepkg@abcd1234</code></li>
    <li>Branch / Tag: <code>refs/tags/v1.0.0</code></li>
  </div>
  <div>
    <p>Publication detail:</p>
    <li>Token Issuer: <code>https://token.actions.githubusercontent.com</code></li>
    <li>Runner Environment: <code>github-hosted</code></li>
    <li>Publication workflow: <code>release.yml@abcd1234</code></li>
    <li>Trigger Event: <code>push</code></li>
  </div>
</div>
</body></html>
`

func TestParseAttestations(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(sampleAttestationHTML))
	if err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}
	fileSection := doc.Find("div[id]").First()

	attestations := parseAttestations(fileSection, "samplepkg", "1.0.0", "samplepkg-1.0.0-py3-none-any.whl")

	if len(attestations) != 1 {
		t.Fatalf("expected 1 attestation, got %d", len(attestations))
	}
	a := attestations[0]
	if a.SigstoreLogIndex != 123456 {
		t.Fatalf("unexpected sigstore_log_index: %d", a.SigstoreLogIndex)
	}
	if a.IntegrationTime != "2026-01-02T03:04:05+0000" {
		t.Fatalf("unexpected integration_time: %q (should read the <time> datetime attribute)", a.IntegrationTime)
	}
	if a.StatementType != "https://in-toto.io/Statement/v1" {
		t.Fatalf("unexpected statement_type: %q", a.StatementType)
	}
	if a.SourceRepo != "github.com/example/samplepkg" {
		t.Fatalf("unexpected source_repo: %q", a.SourceRepo)
	}
	if a.SourceReference != "refs/tags/v1.0.0" {
		t.Fatalf("unexpected source_reference: %q", a.SourceReference)
	}
	if a.TokenIssuer != "https://token.actions.githubusercontent.com" {
		t.Fatalf("unexpected token_issuer: %q", a.TokenIssuer)
	}
	if a.RunnerEnvironment != "github-hosted" {
		t.Fatalf("unexpected runner_environment: %q", a.RunnerEnvironment)
	}
	if a.PublisherWorkflow != "release.yml@abcd1234" {
		t.Fatalf("unexpected publisher_workflow: %q", a.PublisherWorkflow)
	}
	if a.TriggerEvent != "push" {
		t.Fatalf("unexpected trigger_event: %q", a.TriggerEvent)
	}
}

func TestParseAttestations_NoProvenance(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<div id="x"><p>no provenance here</p></div>`))
	if err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}
	fileSection := doc.Find("div[id]").First()

	attestations := parseAttestations(fileSection, "samplepkg", "1.0.0", "x")
	if len(attestations) != 0 {
		t.Fatalf("expected 0 attestations when Provenance section absent, got %d", len(attestations))
	}
}
