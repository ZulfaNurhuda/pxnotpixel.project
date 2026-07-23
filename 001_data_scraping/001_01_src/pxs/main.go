package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pxs/internal/config"
	"pxs/internal/fetch"
	"pxs/internal/parse"
	"pxs/internal/writer"
)

func main() {
	exeDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gagal mendapatkan direktori kerja:", err)
		os.Exit(1)
	}
	listFile := filepath.Join(exeDir, "meta", "top_pypi.txt")
	outputRoot := filepath.Join(exeDir, "..", "..", "001_02_data")

	if err := run(listFile, outputRoot, config.PyPIBaseURL, config.InterPackageDelay); err != nil {
		fmt.Fprintln(os.Stderr, "pxs berhenti karena error:", err)
		os.Exit(1)
	}
	fmt.Println("Selesai. Data hasil scraping sudah ditulis.")
}

// run: baseURL & delay di-inject supaya test bisa pakai httptest server dan tanpa delay.
func run(listFile, outputRoot, baseURL string, delay time.Duration) error {
	names, err := readPackageList(listFile)
	if err != nil {
		return err
	}

	w, err := writer.New()
	if err != nil {
		return fmt.Errorf("menyiapkan writer: %w", err)
	}

	scrapeAll(w, names, baseURL, delay)

	timestamp := time.Now().Format(config.OutputTimestampFormat)
	finalDir := filepath.Join(outputRoot, "pxs_"+timestamp)
	if err := w.Finalize(finalDir); err != nil {
		return fmt.Errorf("finalisasi output gagal: %w", err)
	}

	return nil
}

func readPackageList(listFile string) ([]string, error) {
	data, err := os.ReadFile(listFile)
	if err != nil {
		return nil, fmt.Errorf("meta/top_pypi.txt tidak ditemukan, jalankan bootstrap dulu: %w", err)
	}
	var names []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			names = append(names, line)
		}
	}
	if len(names) < config.NumPackages {
		return nil, fmt.Errorf("meta/top_pypi.txt cuma berisi %d baris, minimal %d dibutuhkan", len(names), config.NumPackages)
	}
	// Kembalikan semua kandidat (bukan cuma NumPackages pertama) sebagai cadangan pengganti bila ada yang gagal di-scrape.
	return names, nil
}

// scrapeAll tidak pernah gagal total karena satu kandidat error (mis. halaman bot-challenge dari pypi.org) — lewati dan ambil cadangan berikutnya.
func scrapeAll(w *writer.Writer, candidates []string, baseURL string, delay time.Duration) {
	target := config.NumPackages
	succeeded := 0
	skipped := 0

	for _, name := range candidates {
		if succeeded >= target {
			break
		}
		if succeeded+skipped > 0 && delay > 0 {
			time.Sleep(delay)
		}
		fmt.Printf("[%d/%d] Mengambil data %s...\n", succeeded+1, target, name)
		if err := scrapePackage(w, name, baseURL); err != nil {
			skipped++
			fmt.Fprintf(os.Stderr, "Lewati %s karena error: %v\n", name, err)
			continue
		}
		succeeded++
	}

	switch {
	case succeeded < target:
		fmt.Printf("Peringatan: cuma berhasil %d dari target %d paket (daftar top_pypi.txt habis, %d dilewati).\n", succeeded, target, skipped)
	case skipped > 0:
		fmt.Printf("Berhasil %d paket (%d dilewati dan digantikan dari cadangan daftar).\n", succeeded, skipped)
	}
}

func scrapePackage(w *writer.Writer, name, baseURL string) error {
	mainURL := fmt.Sprintf("%s/project/%s/", baseURL, name)
	html, err := fetch.Get(mainURL)
	if err != nil {
		return err
	}

	probe, err := parse.ParsePage(html, name, "")
	if err != nil {
		return err
	}
	if len(probe.ReleaseHistory) == 0 {
		return fmt.Errorf("tidak ada riwayat rilis ditemukan untuk %s", name)
	}

	// Halaman utama tanpa versi menampilkan rilis STABLE terbaru, bukan ReleaseHistory[0] yang juga memuat pre-release.
	currentVersion, err := parse.CurrentVersion(html)
	if err != nil {
		return err
	}
	if currentVersion == "" {
		currentVersion = probe.ReleaseHistory[0].Version
	}

	page, err := parse.ParsePage(html, name, currentVersion)
	if err != nil {
		return err
	}
	if err := writePage(w, page, baseURL); err != nil {
		return err
	}

	versionsToFetch := probe.ReleaseHistory
	if len(versionsToFetch) > config.MaxReleasesPerPackage {
		versionsToFetch = versionsToFetch[:config.MaxReleasesPerPackage]
	}
	// Fetch versi historis yang gagal hanya melewatkan baris versi itu, tidak menggagalkan seluruh paket.
	for _, entry := range versionsToFetch {
		if entry.Version == currentVersion {
			continue // sudah diambil lewat halaman utama tanpa versi
		}
		versionURL := fmt.Sprintf("%s/project/%s/%s/", baseURL, name, entry.Version)
		versionHTML, err := fetch.Get(versionURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Lewati versi %s %s karena error: %v\n", name, entry.Version, err)
			continue
		}
		versionPage, err := parse.ParsePage(versionHTML, name, entry.Version)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Lewati versi %s %s karena error: %v\n", name, entry.Version, err)
			continue
		}
		if err := writePage(w, versionPage, baseURL); err != nil {
			return err
		}
	}

	return nil
}

// writePage terima baseURL supaya fetch profil maintainer pakai base yang sama (bukan host hardcoded).
func writePage(w *writer.Writer, page parse.Page, baseURL string) error {
	if err := w.AddPackage(page.Package); err != nil {
		return err
	}
	if page.Organization != nil {
		if err := w.AddOrganization(*page.Organization); err != nil {
			return err
		}
	}
	if err := w.AddRelease(page.Release); err != nil {
		return err
	}
	if err := w.AddReleaseDetail(page.ReleaseDetail); err != nil {
		return err
	}
	for _, keyword := range page.Release.Keywords {
		if err := w.AddReleaseKeyword(page.Release.PackageName, page.Release.Version, keyword); err != nil {
			return err
		}
	}
	for _, extra := range page.Release.ProvidesExtra {
		if err := w.AddReleaseExtra(page.Release.PackageName, page.Release.Version, extra); err != nil {
			return err
		}
	}
	for _, f := range page.Files {
		if err := w.AddReleaseFile(f); err != nil {
			return err
		}
	}
	for _, h := range page.Hashes {
		if err := w.AddFileHash(h); err != nil {
			return err
		}
	}
	for _, t := range page.FileTags {
		if err := w.AddReleaseFileTag(t); err != nil {
			return err
		}
	}
	for _, a := range page.Attestations {
		if err := w.AddAttestation(a); err != nil {
			return err
		}
	}
	for _, l := range page.ProjectLinks {
		if err := w.AddProjectLink(l); err != nil {
			return err
		}
	}
	for _, c := range page.Classifiers {
		if err := w.AddClassifier(c); err != nil {
			return err
		}
	}
	for _, t := range page.Tagged {
		if err := w.AddTaggedWith(t); err != nil {
			return err
		}
	}
	for _, mb := range page.MaintainedBy {
		if err := w.AddMaintainedBy(mb); err != nil {
			return err
		}
	}
	// Fetch profil maintainer yang gagal hanya melewatkan maintainer itu, tidak digagalkan; beda kasus dari "Date joined" yang memang tidak ada (bukan error).
	for _, username := range page.Maintainers {
		if w.HasMaintainer(username) {
			continue
		}
		profileHTML, err := fetch.Get(fmt.Sprintf("%s/user/%s/", baseURL, username))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Lewati profil maintainer %s karena error: %v\n", username, err)
			w.MarkMaintainerAttempted(username)
			continue
		}
		joinedAt, err := parse.ParseMaintainerProfile(profileHTML)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Lewati profil maintainer %s karena error: %v\n", username, err)
			w.MarkMaintainerAttempted(username)
			continue
		}
		if err := w.AddMaintainer(parse.Maintainer{Username: username, JoinedAt: joinedAt}); err != nil {
			return err
		}
	}
	return nil
}
