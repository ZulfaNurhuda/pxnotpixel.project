package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"pxs/internal/parse"
)

const testMainPageHTML = `
<html><body>
<div class="package-header__date">Released: Jan 2, 2026</div>
<p class="package-description__summary">Test package.</p>
<div class="project-description"><p>Desc</p></div>
<div class="release-timeline">
  <div class="release"><div class="release__version">1.0.0</div></div>
</div>
</body></html>
`

func TestRun_WritesOutputForOnePackage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testMainPageHTML))
	}))
	defer server.Close()

	listFile := filepath.Join(t.TempDir(), "top_pypi.txt")
	if err := os.WriteFile(listFile, []byte("testpkg\n"+strings.Repeat("filler\n", 99)), 0o644); err != nil {
		t.Fatalf("writing package list: %v", err)
	}

	outputRoot := t.TempDir()

	if err := run(listFile, outputRoot, server.URL, 0); err != nil {
		t.Fatalf("run() error: %v", err)
	}

	entries, err := os.ReadDir(outputRoot)
	if err != nil {
		t.Fatalf("reading output root: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 run folder, got %d", len(entries))
	}

	releaseData, err := os.ReadFile(filepath.Join(outputRoot, entries[0].Name(), "release.json"))
	if err != nil {
		t.Fatalf("reading release.json: %v", err)
	}
	if !strings.Contains(string(releaseData), `"package_name":"testpkg"`) {
		t.Fatalf("release.json missing expected package_name: %s", releaseData)
	}
}

// testMultiReleaseHTML: 3 rilis (2.0.0 biasa, 1.9.0 pre-release, 1.8.0 yanked), disajikan untuk semua path seperti pypi.org asli.
const testMultiReleaseHTML = `
<html><body>
<div class="package-header__date">Released: Jan 2, 2026</div>
<p class="package-description__summary">Test package.</p>
<div class="project-description"><p>Desc</p></div>
<div class="release-timeline">
  <div class="release"><div class="release__version">2.0.0</div></div>
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

func TestRun_MultipleReleases_NoDuplicatePackageRows_AndYankedFlagsPropagate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testMultiReleaseHTML))
	}))
	defer server.Close()

	listFile := filepath.Join(t.TempDir(), "top_pypi.txt")
	if err := os.WriteFile(listFile, []byte("testpkg\n"+strings.Repeat("filler\n", 99)), 0o644); err != nil {
		t.Fatalf("writing package list: %v", err)
	}

	outputRoot := t.TempDir()

	if err := run(listFile, outputRoot, server.URL, 0); err != nil {
		t.Fatalf("run() error: %v", err)
	}

	entries, err := os.ReadDir(outputRoot)
	if err != nil {
		t.Fatalf("reading output root: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 run folder, got %d", len(entries))
	}
	runDir := filepath.Join(outputRoot, entries[0].Name())

	// (a) baris rilis yanked harus punya yanked=true dan yanked_reason terisi.
	releaseData, err := os.ReadFile(filepath.Join(runDir, "release.json"))
	if err != nil {
		t.Fatalf("reading release.json: %v", err)
	}
	var releases []parse.Release
	if err := json.Unmarshal(releaseData, &releases); err != nil {
		t.Fatalf("decoding release.json: %v", err)
	}
	var found bool
	for _, r := range releases {
		if r.Version != "1.8.0" {
			continue
		}
		found = true
		if !r.Yanked {
			t.Errorf("expected 1.8.0 release to be yanked, got Yanked=%v", r.Yanked)
		}
		if r.YankedReason == "" {
			t.Errorf("expected 1.8.0 release to have a non-empty yanked_reason")
		}
	}
	if !found {
		t.Fatalf("release.json has no row for version 1.8.0: %s", releaseData)
	}

	// (b) package.json harus punya tepat 1 baris per nama package, bukan 1 per rilis.
	packageData, err := os.ReadFile(filepath.Join(runDir, "package.json"))
	if err != nil {
		t.Fatalf("reading package.json: %v", err)
	}
	var packages []parse.Package
	if err := json.Unmarshal(packageData, &packages); err != nil {
		t.Fatalf("decoding package.json: %v", err)
	}
	counts := map[string]int{}
	for _, p := range packages {
		counts[p.Name]++
	}
	if counts["testpkg"] != 1 {
		t.Errorf("expected exactly 1 package.json row for testpkg, got %d: %s", counts["testpkg"], packageData)
	}
	if counts["filler"] != 1 {
		t.Errorf("expected exactly 1 package.json row for filler, got %d: %s", counts["filler"], packageData)
	}

	// (c) dedup maintained_by.json tidak diuji di sini (fixture tanpa sidebar Maintainers).
}

// TestRun_SkipsFailingPackageAndContinues: satu package gagal parse (mis. halaman bot-challenge) tidak boleh menggagalkan seluruh run.
func TestRun_SkipsFailingPackageAndContinues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/blockedpkg/") {
			// Simulasi halaman bot-challenge: 200 OK tapi tanpa release-timeline, jadi ReleaseHistory kosong.
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><body><p>Client Challenge</p></body></html>`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testMainPageHTML))
	}))
	defer server.Close()

	names := "goodpkg1\nblockedpkg\ngoodpkg2\n" + strings.Repeat("filler\n", 97)
	listFile := filepath.Join(t.TempDir(), "top_pypi.txt")
	if err := os.WriteFile(listFile, []byte(names), 0o644); err != nil {
		t.Fatalf("writing package list: %v", err)
	}

	outputRoot := t.TempDir()

	if err := run(listFile, outputRoot, server.URL, 0); err != nil {
		t.Fatalf("run() should not fail the whole run over one blocked package, got error: %v", err)
	}

	entries, err := os.ReadDir(outputRoot)
	if err != nil {
		t.Fatalf("reading output root: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 run folder, got %d", len(entries))
	}
	runDir := filepath.Join(outputRoot, entries[0].Name())

	packageData, err := os.ReadFile(filepath.Join(runDir, "package.json"))
	if err != nil {
		t.Fatalf("reading package.json: %v", err)
	}
	var packages []parse.Package
	if err := json.Unmarshal(packageData, &packages); err != nil {
		t.Fatalf("decoding package.json: %v", err)
	}
	names_ := map[string]bool{}
	for _, p := range packages {
		names_[p.Name] = true
	}
	if !names_["goodpkg1"] || !names_["goodpkg2"] {
		t.Fatalf("expected goodpkg1 and goodpkg2 to still be scraped, got: %s", packageData)
	}
	if names_["blockedpkg"] {
		t.Fatalf("blockedpkg should have been skipped (it never produced valid data), but found it in package.json: %s", packageData)
	}
}

// TestRun_BackfillsFromReserveToReachTargetCount: hitungan akhir harus tetap capai config.NumPackages, ambil cadangan bila ada yang gagal.
func TestRun_BackfillsFromReserveToReachTargetCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/blockedpkg/") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><body><p>Client Challenge</p></body></html>`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testMainPageHTML))
	}))
	defer server.Close()

	// 102 kandidat: goodpkg1, blockedpkg (gagal), lalu filler1..filler100 sebagai cadangan (nama unik agar bisa dihitung).
	var sb strings.Builder
	sb.WriteString("goodpkg1\nblockedpkg\n")
	for i := 1; i <= 100; i++ {
		fmt.Fprintf(&sb, "filler%d\n", i)
	}

	listFile := filepath.Join(t.TempDir(), "top_pypi.txt")
	if err := os.WriteFile(listFile, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("writing package list: %v", err)
	}

	outputRoot := t.TempDir()

	if err := run(listFile, outputRoot, server.URL, 0); err != nil {
		t.Fatalf("run() error: %v", err)
	}

	entries, err := os.ReadDir(outputRoot)
	if err != nil {
		t.Fatalf("reading output root: %v", err)
	}
	runDir := filepath.Join(outputRoot, entries[0].Name())

	packageData, err := os.ReadFile(filepath.Join(runDir, "package.json"))
	if err != nil {
		t.Fatalf("reading package.json: %v", err)
	}
	var packages []parse.Package
	if err := json.Unmarshal(packageData, &packages); err != nil {
		t.Fatalf("decoding package.json: %v", err)
	}

	if len(packages) != 100 {
		t.Fatalf("expected exactly 100 scraped packages (backfilled from reserve), got %d: %s", len(packages), packageData)
	}

	scraped := map[string]bool{}
	for _, p := range packages {
		scraped[p.Name] = true
	}
	if !scraped["goodpkg1"] {
		t.Error("expected goodpkg1 to be scraped")
	}
	if scraped["blockedpkg"] {
		t.Error("blockedpkg should never appear in output, it always fails")
	}
	// goodpkg1 + filler1..filler99 = 100 sukses; filler100 tidak pernah dicapai karena loop berhenti tepat di target.
	for i := 1; i <= 99; i++ {
		if !scraped[fmt.Sprintf("filler%d", i)] {
			t.Errorf("expected filler%d to be scraped as backfill", i)
		}
	}
	if scraped["filler100"] {
		t.Error("filler100 should never have been reached — loop should stop right at 100 successes")
	}
}

// testTwoReleaseHTML: 2 rilis (2.0.0, 1.9.0) untuk memicu satu fetch versi historis.
const testTwoReleaseHTML = `
<html><body>
<div class="package-header__date">Released: Jan 2, 2026</div>
<p class="package-description__summary">Test package.</p>
<div class="project-description"><p>Desc</p></div>
<div class="release-timeline">
  <div class="release"><div class="release__version">2.0.0</div></div>
  <div class="release"><div class="release__version">1.9.0</div></div>
</div>
</body></html>
`

// TestRun_SkipsFailedVersionFetchButKeepsPackageAndOtherVersions: ditemukan di live pypi.org, ~45% baris rilis historis rusak diam-diam bila satu halaman versi gagal ikut menggagalkan package.
func TestRun_SkipsFailedVersionFetchButKeepsPackageAndOtherVersions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/testpkg/1.9.0/") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><head><title>Client Challenge</title></head><body></body></html>`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testTwoReleaseHTML))
	}))
	defer server.Close()

	names := "testpkg\n" + strings.Repeat("filler\n", 99)
	listFile := filepath.Join(t.TempDir(), "top_pypi.txt")
	if err := os.WriteFile(listFile, []byte(names), 0o644); err != nil {
		t.Fatalf("writing package list: %v", err)
	}

	outputRoot := t.TempDir()

	if err := run(listFile, outputRoot, server.URL, 0); err != nil {
		t.Fatalf("run() should not fail over one blocked historical version, got error: %v", err)
	}

	entries, err := os.ReadDir(outputRoot)
	if err != nil {
		t.Fatalf("reading output root: %v", err)
	}
	runDir := filepath.Join(outputRoot, entries[0].Name())

	packageData, err := os.ReadFile(filepath.Join(runDir, "package.json"))
	if err != nil {
		t.Fatalf("reading package.json: %v", err)
	}
	var packages []parse.Package
	if err := json.Unmarshal(packageData, &packages); err != nil {
		t.Fatalf("decoding package.json: %v", err)
	}
	found := false
	for _, p := range packages {
		if p.Name == "testpkg" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected testpkg to still be scraped despite one blocked version, package.json: %s", packageData)
	}

	releaseData, err := os.ReadFile(filepath.Join(runDir, "release.json"))
	if err != nil {
		t.Fatalf("reading release.json: %v", err)
	}
	var releases []parse.Release
	if err := json.Unmarshal(releaseData, &releases); err != nil {
		t.Fatalf("decoding release.json: %v", err)
	}
	var testpkgVersions []string
	for _, r := range releases {
		if r.PackageName == "testpkg" {
			testpkgVersions = append(testpkgVersions, r.Version)
		}
	}
	if len(testpkgVersions) != 1 || testpkgVersions[0] != "2.0.0" {
		t.Fatalf("expected only version 2.0.0 written for testpkg (1.9.0 blocked and skipped), got: %v", testpkgVersions)
	}
}

// TestRun_SkipsFailedMaintainerProfileFetchButKeepsPackage: profil maintainer yang diblokir tidak boleh menghasilkan baris joined_at kosong palsu — maintainer dilewati saja.
func TestRun_SkipsFailedMaintainerProfileFetchButKeepsPackage(t *testing.T) {
	const htmlWithMaintainer = `
<html><body>
<div class="package-header__date">Released: Jan 2, 2026</div>
<p class="package-description__summary">Test package.</p>
<div class="project-description"><p>Desc</p></div>
<div class="sidebar-section verified">
  <h6>Maintainers</h6>
  <span class="sidebar-section__user-gravatar-text">somebody</span>
</div>
<div class="release-timeline">
  <div class="release"><div class="release__version">1.0.0</div></div>
</div>
</body></html>
`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/user/somebody/") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><head><title>Client Challenge</title></head><body></body></html>`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(htmlWithMaintainer))
	}))
	defer server.Close()

	names := "testpkg\n" + strings.Repeat("filler\n", 99)
	listFile := filepath.Join(t.TempDir(), "top_pypi.txt")
	if err := os.WriteFile(listFile, []byte(names), 0o644); err != nil {
		t.Fatalf("writing package list: %v", err)
	}

	outputRoot := t.TempDir()

	if err := run(listFile, outputRoot, server.URL, 0); err != nil {
		t.Fatalf("run() should not fail over one blocked maintainer profile, got error: %v", err)
	}

	entries, err := os.ReadDir(outputRoot)
	if err != nil {
		t.Fatalf("reading output root: %v", err)
	}
	runDir := filepath.Join(outputRoot, entries[0].Name())

	packageData, err := os.ReadFile(filepath.Join(runDir, "package.json"))
	if err != nil {
		t.Fatalf("reading package.json: %v", err)
	}
	var packages []parse.Package
	if err := json.Unmarshal(packageData, &packages); err != nil {
		t.Fatalf("decoding package.json: %v", err)
	}
	found := false
	for _, p := range packages {
		if p.Name == "testpkg" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected testpkg to still be scraped despite blocked maintainer profile, package.json: %s", packageData)
	}

	maintainerData, err := os.ReadFile(filepath.Join(runDir, "maintainer.json"))
	if err != nil {
		t.Fatalf("reading maintainer.json: %v", err)
	}
	var maintainers []parse.Maintainer
	if err := json.Unmarshal(maintainerData, &maintainers); err != nil {
		t.Fatalf("decoding maintainer.json: %v", err)
	}
	for _, m := range maintainers {
		if m.Username == "somebody" {
			t.Fatalf("expected 'somebody' to be absent from maintainer.json (blocked profile fetch, not a false-empty joined_at row), got: %+v", m)
		}
	}
}

// TestRun_FailedMaintainerProfileFetchIsOnlyAttemptedOnce: maintainer muncul di tiap halaman versi, jadi profil gagal harus diingat, bukan dicoba ulang tiap versi.
func TestRun_FailedMaintainerProfileFetchIsOnlyAttemptedOnce(t *testing.T) {
	const htmlWithMaintainerAndHistory = `
<html><body>
<div class="package-header__date">Released: Jan 2, 2026</div>
<p class="package-description__summary">Test package.</p>
<div class="project-description"><p>Desc</p></div>
<div class="sidebar-section verified">
  <h6>Maintainers</h6>
  <span class="sidebar-section__user-gravatar-text">somebody</span>
</div>
<div class="release-timeline">
  <div class="release"><div class="release__version">3.0.0</div></div>
  <div class="release"><div class="release__version">2.0.0</div></div>
  <div class="release"><div class="release__version">1.0.0</div></div>
</div>
</body></html>
`
	var mu sync.Mutex
	profileHits := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/user/somebody/") {
			mu.Lock()
			profileHits++
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><head><title>Client Challenge</title></head><body></body></html>`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(htmlWithMaintainerAndHistory))
	}))
	defer server.Close()

	names := "testpkg\n" + strings.Repeat("filler\n", 99)
	listFile := filepath.Join(t.TempDir(), "top_pypi.txt")
	if err := os.WriteFile(listFile, []byte(names), 0o644); err != nil {
		t.Fatalf("writing package list: %v", err)
	}

	outputRoot := t.TempDir()

	if err := run(listFile, outputRoot, server.URL, 0); err != nil {
		t.Fatalf("run() error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if profileHits != 1 {
		t.Fatalf("expected exactly 1 profile fetch attempt for 'somebody' across testpkg's 3 versions, got %d", profileHits)
	}
}

// TestRun_UnversionedMainPageShowsStableNotLatestPrerelease: ditemukan di pypi.org/project/pydantic/ — halaman utama tanpa versi menampilkan rilis STABLE, bukan pre-release terbaru di sidebar; anggap sama = data pre-release tidak pernah benar-benar diambil.
func TestRun_UnversionedMainPageShowsStableNotLatestPrerelease(t *testing.T) {
	const releaseTimeline = `
<div class="release-timeline">
  <div class="release">
    <div class="release__version">3.0.0 <span class="badge--warning">pre-release</span></div>
  </div>
  <div class="release"><div class="release__version">2.0.0</div></div>
  <div class="release"><div class="release__version">1.0.0</div></div>
</div>
`
	mainHTML := `
<html><body>
<h1 class="package-header__name">testpkg 2.0.0</h1>
<div class="package-header__date">Released: Jan 2, 2026</div>
<p class="package-description__summary">Stable release summary.</p>
<div class="project-description"><p>Desc</p></div>
` + releaseTimeline + `
</body></html>
`
	prereleaseHTML := `
<html><body>
<h1 class="package-header__name">testpkg 3.0.0</h1>
<div class="package-header__date">Released: Jan 3, 2026</div>
<p class="package-description__summary">Pre-release summary.</p>
<div class="project-description"><p>Desc</p></div>
` + releaseTimeline + `
</body></html>
`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if strings.Contains(r.URL.Path, "/testpkg/3.0.0/") {
			w.Write([]byte(prereleaseHTML))
			return
		}
		// halaman utama dan /2.0.0/, /1.0.0/ sama-sama pakai fixture "stable" — hanya label versi dari scrapePackage yang membedakan.
		w.Write([]byte(mainHTML))
	}))
	defer server.Close()

	names := "testpkg\n" + strings.Repeat("filler\n", 99)
	listFile := filepath.Join(t.TempDir(), "top_pypi.txt")
	if err := os.WriteFile(listFile, []byte(names), 0o644); err != nil {
		t.Fatalf("writing package list: %v", err)
	}

	outputRoot := t.TempDir()

	if err := run(listFile, outputRoot, server.URL, 0); err != nil {
		t.Fatalf("run() error: %v", err)
	}

	entries, err := os.ReadDir(outputRoot)
	if err != nil {
		t.Fatalf("reading output root: %v", err)
	}
	runDir := filepath.Join(outputRoot, entries[0].Name())

	releaseData, err := os.ReadFile(filepath.Join(runDir, "release.json"))
	if err != nil {
		t.Fatalf("reading release.json: %v", err)
	}
	var releases []parse.Release
	if err := json.Unmarshal(releaseData, &releases); err != nil {
		t.Fatalf("decoding release.json: %v", err)
	}

	byVersion := map[string][]parse.Release{}
	for _, r := range releases {
		if r.PackageName == "testpkg" {
			byVersion[r.Version] = append(byVersion[r.Version], r)
		}
	}

	for _, v := range []string{"3.0.0", "2.0.0", "1.0.0"} {
		if len(byVersion[v]) != 1 {
			t.Errorf("expected exactly 1 release.json row for testpkg %s, got %d", v, len(byVersion[v]))
		}
	}

	if got := byVersion["2.0.0"][0].Summary; got != "Stable release summary." {
		t.Errorf("expected 2.0.0's row to carry the stable page's own summary, got %q", got)
	}
	if got := byVersion["3.0.0"][0].Summary; got != "Pre-release summary." {
		t.Errorf("expected 3.0.0 to be fetched via its own page (not mislabeled from the main page), got summary %q", got)
	}
}
