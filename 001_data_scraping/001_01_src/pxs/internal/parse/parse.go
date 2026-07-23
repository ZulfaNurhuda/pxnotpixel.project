package parse

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Package: pakai natural key organisasi, bukan surrogate org_id.
type Package struct {
	Name              string `json:"name"`
	LifecycleStatus   string `json:"lifecycle_status,omitempty"`
	OrganizationOwner string `json:"organization_owner,omitempty"`
}

type Organization struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

func orgSlugFromHref(href string) string {
	href = strings.TrimSuffix(href, "/")
	parts := strings.Split(href, "/")
	return parts[len(parts)-1]
}

// pypi.org merender sidebar ini dua kali (tab visible + salinan tersembunyi untuk tab-switcher); tanpa .First() tiap baris hasil extract jadi 2x lipat.
func verifiedSection(doc *goquery.Document) *goquery.Selection {
	return doc.Find(".sidebar-section.verified").First()
}

func unverifiedSection(doc *goquery.Document) *goquery.Selection {
	return doc.Find(".sidebar-section.unverified").First()
}

func parsePackageAndOrg(doc *goquery.Document) (Package, *Organization) {
	var pkg Package

	doc.Find(".package-header__right .status-badge span").Each(func(_ int, s *goquery.Selection) {
		if strings.TrimSpace(s.Text()) == "This project has been quarantined" {
			pkg.LifecycleStatus = "quarantined"
		}
	})

	var org *Organization
	verifiedSection(doc).Find("h6").Each(func(_ int, h6 *goquery.Selection) {
		if strings.TrimSpace(h6.Text()) != "Owner" {
			return
		}
		link := h6.NextFiltered("ul.vertical-tabs__list").Find("li a").First()
		href, _ := link.Attr("href")
		if href == "" {
			return
		}
		slug := orgSlugFromHref(href)
		org = &Organization{
			Name:        slug,
			DisplayName: strings.TrimSpace(link.Text()),
		}
		pkg.OrganizationOwner = slug
	})

	return pkg, org
}

// Release: field description/author dipisah ke ReleaseDetail (vertical partitioning).
type Release struct {
	PackageName     string   `json:"package_name"`
	Version         string   `json:"version"`
	Created         string   `json:"created,omitempty"`
	IsPrerelease    bool     `json:"is_prerelease"`
	Yanked          bool     `json:"yanked"`
	LifecycleStatus string   `json:"lifecycle_status,omitempty"`
	YankedReason    string   `json:"yanked_reason,omitempty"`
	Summary         string   `json:"summary,omitempty"`
	License         string   `json:"license,omitempty"`
	RequiresPython  string   `json:"requires_python,omitempty"`
	Keywords        []string `json:"-"` // ditulis terpisah ke release_keyword.json
	ProvidesExtra   []string `json:"-"` // ditulis terpisah ke release_extra.json
}

type ReleaseDetail struct {
	PackageName                 string `json:"package_name"`
	Version                     string `json:"version"`
	Description                 string `json:"description,omitempty"`
	MetaAuthor                  string `json:"meta_author,omitempty"`
	MetaAuthorEmail              string `json:"meta_author_email,omitempty"`
	MetaAuthorEmailVerified      bool   `json:"meta_author_email_verified"`
	MetaMaintainer               string `json:"meta_maintainer,omitempty"`
	MetaMaintainerEmail          string `json:"meta_maintainer_email,omitempty"`
	MetaMaintainerEmailVerified  bool   `json:"meta_maintainer_email_verified"`
}

func findMetaLine(section *goquery.Selection, label string) (line *goquery.Selection, ok bool) {
	var found *goquery.Selection
	section.Find("li").EachWithBreak(func(_ int, li *goquery.Selection) bool {
		if strings.TrimSpace(li.Find("strong").First().Text()) == label {
			found = li
			return false
		}
		return true
	})
	if found == nil {
		return nil, false
	}
	return found, true
}

func metaEmail(li *goquery.Selection) string {
	href, _ := li.Find("a[href^='mailto:']").Attr("href")
	return strings.TrimPrefix(href, "mailto:")
}

// extractMetaValue: TrimPrefix jalan di teks yang sudah di-trim (markup asli berindentasi), lalu dipotong di newline pertama (buang footer SPDX di License Expression).
func extractMetaValue(li *goquery.Selection, label string) string {
	text := strings.TrimPrefix(strings.TrimSpace(li.Text()), label)
	if idx := strings.IndexByte(text, '\n'); idx != -1 {
		text = text[:idx]
	}
	return strings.TrimSpace(text)
}

func parseReleaseMeta(doc *goquery.Document, packageName, version string) (Release, ReleaseDetail) {
	release := Release{PackageName: packageName, Version: version}
	detail := ReleaseDetail{PackageName: packageName, Version: version}

	// Baca atribut datetime (ISO 8601), bukan teks tampilan yang di-localize client dan bisa beda hari dari waktu UTC aslinya.
	release.Created, _ = doc.Find(".package-header__date time").First().Attr("datetime")
	release.Summary = strings.TrimSpace(doc.Find(".package-description__summary").First().Text())

	if html, err := doc.Find(".project-description").First().Html(); err == nil {
		detail.Description = strings.TrimSpace(html)
	}

	unverified := unverifiedSection(doc)
	verified := verifiedSection(doc)

	if li, ok := findMetaLine(unverified, "License Expression:"); ok {
		release.License = extractMetaValue(li, "License Expression:")
	} else if li, ok := findMetaLine(unverified, "License:"); ok {
		release.License = extractMetaValue(li, "License:")
	}

	if li, ok := findMetaLine(unverified, "Requires:"); ok {
		text := extractMetaValue(li, "Requires:")
		release.RequiresPython = strings.TrimSpace(strings.TrimPrefix(text, "Python"))
	}

	if li, ok := findMetaLine(verified, "Author:"); ok {
		detail.MetaAuthor = extractMetaValue(li, "Author:")
		detail.MetaAuthorEmail = metaEmail(li)
		detail.MetaAuthorEmailVerified = true
	} else if li, ok := findMetaLine(unverified, "Author:"); ok {
		detail.MetaAuthor = extractMetaValue(li, "Author:")
		detail.MetaAuthorEmail = metaEmail(li)
	}

	if li, ok := findMetaLine(verified, "Maintainer:"); ok {
		detail.MetaMaintainer = extractMetaValue(li, "Maintainer:")
		detail.MetaMaintainerEmail = metaEmail(li)
		detail.MetaMaintainerEmailVerified = true
	} else if li, ok := findMetaLine(unverified, "Maintainer:"); ok {
		detail.MetaMaintainer = extractMetaValue(li, "Maintainer:")
		detail.MetaMaintainerEmail = metaEmail(li)
	}

	unverified.Find("li.tags .package-keyword").Each(func(_ int, s *goquery.Selection) {
		// Koma trailing muncul SETELAH newline di dalam span ("filepost\n,"), jadi butuh Trim bukan TrimSuffix(",").
		kw := strings.Trim(s.Text(), " \n\t\r,")
		if kw != "" {
			release.Keywords = append(release.Keywords, kw)
		}
	})

	if li, ok := findMetaLine(unverified, "Provides-Extra:"); ok {
		li.Find("code").Each(func(_ int, s *goquery.Selection) {
			release.ProvidesExtra = append(release.ProvidesExtra, strings.TrimSpace(s.Text()))
		})
	}

	return release, detail
}

// ReleaseHistoryEntry: satu baris dari tab "Release history", dipakai memilih versi yang di-scrape dan status is_prerelease/yanked.
type ReleaseHistoryEntry struct {
	Version      string
	IsPrerelease bool
	Yanked       bool
	YankedReason string
}

func parseReleaseHistory(doc *goquery.Document) []ReleaseHistoryEntry {
	var entries []ReleaseHistoryEntry
	doc.Find(".release-timeline .release").Each(func(_ int, item *goquery.Selection) {
		versionEl := item.Find(".release__version").First()
		version := strings.TrimSpace(versionEl.Contents().Not("span").Text())

		entry := ReleaseHistoryEntry{Version: version}
		versionEl.Find(".badge--warning").Each(func(_ int, s *goquery.Selection) {
			if strings.TrimSpace(s.Text()) == "pre-release" {
				entry.IsPrerelease = true
			}
		})
		versionEl.Find(".badge--danger").Each(func(_ int, s *goquery.Selection) {
			if strings.TrimSpace(s.Text()) == "yanked" {
				entry.Yanked = true
			}
		})
		if entry.Yanked {
			entry.YankedReason = strings.TrimSpace(item.Find(".release__yanked-reason").First().Text())
		}
		entries = append(entries, entry)
	})
	return entries
}

type ReleaseFile struct {
	PackageName         string `json:"package_name"`
	Version             string `json:"version"`
	Filename            string `json:"filename"`
	Path                string `json:"path"`
	Size                int64  `json:"size"`
	UploadTime          string `json:"upload_time"`
	IsTrustedPublishing bool   `json:"is_trusted_publishing"`
	UploadedVia         string `json:"uploaded_via,omitempty"`
	PackageType         string `json:"packagetype"`
}

type FileHash struct {
	PackageName string `json:"package_name"`
	Version     string `json:"version"`
	Filename    string `json:"filename"`
	Algorithm   string `json:"algorithm"`
	Digest      string `json:"digest"`
}

type ReleaseFileTag struct {
	PackageName string `json:"package_name"`
	Version     string `json:"version"`
	Filename    string `json:"filename"`
	WheelTag    string `json:"wheel_tag"`
}

func findLiPrefix(root *goquery.Selection, prefix string) (string, bool) {
	var value string
	var found bool
	root.Find("li").EachWithBreak(func(_ int, li *goquery.Selection) bool {
		text := strings.TrimSpace(li.Text())
		if strings.HasPrefix(text, prefix) {
			value = strings.TrimSpace(strings.TrimPrefix(text, prefix))
			found = true
			return false
		}
		return true
	})
	return value, found
}

type Attestation struct {
	PackageName       string `json:"package_name"`
	Version           string `json:"version"`
	Filename          string `json:"filename"`
	SigstoreLogIndex  int64  `json:"sigstore_log_index"`
	IntegrationTime   string `json:"integration_time"`
	StatementType     string `json:"statement_type"`
	PredicateType     string `json:"predicate_type"`
	SubjectName       string `json:"subject_name"`
	SubjectDigest     string `json:"subject_digest"`
	SourceRepo        string `json:"source_repo,omitempty"`
	SourceReference   string `json:"source_reference,omitempty"`
	TokenIssuer       string `json:"token_issuer"`
	RunnerEnvironment string `json:"runner_environment,omitempty"`
	PublisherWorkflow string `json:"publisher_workflow,omitempty"`
	TriggerEvent      string `json:"trigger_event,omitempty"`
}

// liTimeAttr: ambil atribut datetime (ISO 8601), bukan teks tampilan yang client-localized.
func liTimeAttr(root *goquery.Selection, prefix string) string {
	var value string
	root.Find("li").EachWithBreak(func(_ int, li *goquery.Selection) bool {
		if !strings.HasPrefix(strings.TrimSpace(li.Text()), prefix) {
			return true
		}
		value, _ = li.Find("time").First().Attr("datetime")
		return false
	})
	return value
}

func liCodeValue(root *goquery.Selection, prefix string) string {
	var value string
	root.Find("li").EachWithBreak(func(_ int, li *goquery.Selection) bool {
		if !strings.HasPrefix(strings.TrimSpace(li.Text()), prefix) {
			return true
		}
		value = strings.TrimSpace(li.Find("code").First().Text())
		return false
	})
	return value
}

// parseAttestations: kembalikan slice kosong bila file tidak punya section Provenance (kasus normal).
func parseAttestations(fileSection *goquery.Selection, packageName, version, filename string) []Attestation {
	hasProvenance := false
	fileSection.Find("h4, h3").Each(func(_ int, h *goquery.Selection) {
		if strings.TrimSpace(h.Text()) == "Provenance" {
			hasProvenance = true
		}
	})
	if !hasProvenance {
		return nil
	}

	a := Attestation{PackageName: packageName, Version: version, Filename: filename}

	fileSection.Find("li").EachWithBreak(func(_ int, li *goquery.Selection) bool {
		text := strings.TrimSpace(li.Text())
		if strings.HasPrefix(text, "Sigstore transparency entry:") {
			href, _ := li.Find("a").Attr("href")
			if idx := strings.Index(href, "logIndex="); idx != -1 {
				fmt.Sscanf(href[idx+len("logIndex="):], "%d", &a.SigstoreLogIndex)
			}
		}
		return true
	})
	a.IntegrationTime = liTimeAttr(fileSection, "Sigstore integration time:")

	a.StatementType = liCodeValue(fileSection, "Statement type:")
	a.PredicateType = liCodeValue(fileSection, "Predicate type:")
	a.SubjectName = liCodeValue(fileSection, "Subject name:")
	a.SubjectDigest = liCodeValue(fileSection, "Subject digest:")

	if permalink := liCodeValue(fileSection, "Permalink:"); permalink != "" {
		a.SourceRepo = strings.SplitN(permalink, "@", 2)[0]
	}
	a.SourceReference = liCodeValue(fileSection, "Branch / Tag:")
	a.TokenIssuer = liCodeValue(fileSection, "Token Issuer:")
	a.RunnerEnvironment = liCodeValue(fileSection, "Runner Environment:")
	a.PublisherWorkflow = liCodeValue(fileSection, "Publication workflow:")
	a.TriggerEvent = liCodeValue(fileSection, "Trigger Event:")

	return []Attestation{a}
}

// parseFileTypeSummary: klasifikasi sdist/wheel cuma bisa didapat dari listing #files, bukan dari detail panel file itu sendiri.
func parseFileTypeSummary(doc *goquery.Document) map[string]string {
	result := make(map[string]string)
	packageType := ""
	doc.Find("#files").Children().Each(func(_ int, child *goquery.Selection) {
		if goquery.NodeName(child) == "h3" {
			switch strings.TrimSpace(child.Text()) {
			case "Source Distribution":
				packageType = "sdist"
			case "Built Distribution":
				packageType = "wheel"
			}
			return
		}
		if packageType == "" || !child.HasClass("file") {
			return
		}
		a := child.Find("a[href^='https://files.pythonhosted.org/']").First()
		filename := strings.TrimSpace(a.Text())
		if filename != "" {
			result[filename] = packageType
		}
	})
	return result
}

// parseFileSize: parse teks ukuran human-readable ("228.1 kB") jadi bytes; unit tak dikenal fallback ke multiplier 1.
func parseFileSize(text string) int64 {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return 0
	}
	value, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	multiplier := float64(1)
	if len(fields) > 1 {
		switch fields[1] {
		case "kB", "KB":
			multiplier = 1024
		case "MB":
			multiplier = 1024 * 1024
		case "GB":
			multiplier = 1024 * 1024 * 1024
		}
	}
	return int64(value * multiplier)
}

// parseReleaseFiles: detail panel file adalah div[id=<filename>] sibling dari #files (bukan nested); div[id] lain di halaman dilewati.
func parseReleaseFiles(doc *goquery.Document, packageName, version string) ([]ReleaseFile, []FileHash, []ReleaseFileTag, []Attestation) {
	var files []ReleaseFile
	var hashes []FileHash
	var tags []ReleaseFileTag
	var attestations []Attestation

	packageTypeByFilename := parseFileTypeSummary(doc)

	doc.Find("div[id]").Each(func(_ int, s *goquery.Selection) {
		filename, _ := s.Attr("id")
		packageType, isFile := packageTypeByFilename[filename]
		if !isFile {
			return
		}

		href, _ := s.Find("a[href]").First().Attr("href")

		sizeText, _ := findLiPrefix(s, "Size:")
		size := parseFileSize(sizeText)

		uploadTime := liTimeAttr(s, "Upload date:")
		trustedText, _ := findLiPrefix(s, "Uploaded using Trusted Publishing?")
		uploadedVia, _ := findLiPrefix(s, "Uploaded via:")

		files = append(files, ReleaseFile{
			PackageName:         packageName,
			Version:             version,
			Filename:            filename,
			Path:                href,
			Size:                size,
			UploadTime:          uploadTime,
			IsTrustedPublishing: trustedText == "Yes",
			UploadedVia:         uploadedVia,
			PackageType:         packageType,
		})

		s.Find("table.table--hashes tbody tr").Each(func(_ int, row *goquery.Selection) {
			algorithm := strings.TrimSpace(row.Find("th[scope='row']").Text())
			digest := strings.TrimSpace(row.Find("td code").First().Text())
			hashes = append(hashes, FileHash{
				PackageName: packageName,
				Version:     version,
				Filename:    filename,
				Algorithm:   algorithm,
				Digest:      digest,
			})
		})

		// "Tags:" cuma deskriptor tunggal ("Python 3"/"Source"), bukan tag triple PEP 425 — jadi cuma wheel yang dapat baris di sini.
		if packageType == "wheel" {
			if tagText, ok := findLiPrefix(s, "Tags:"); ok {
				tagText = strings.TrimSpace(tagText)
				if tagText != "" {
					tags = append(tags, ReleaseFileTag{
						PackageName: packageName,
						Version:     version,
						Filename:    filename,
						WheelTag:    tagText,
					})
				}
			}
		}

		attestations = append(attestations, parseAttestations(s, packageName, version, filename)...)
	})

	return files, hashes, tags, attestations
}

type ProjectLink struct {
	PackageName string `json:"package_name"`
	Version     string `json:"version"`
	Label       string `json:"label"`
	URL         string `json:"url"`
	Verified    bool   `json:"verified"`
}

func parseProjectLinksIn(section *goquery.Selection, verified bool, packageName, version string) []ProjectLink {
	var links []ProjectLink
	section.Find("h6").Each(func(_ int, h6 *goquery.Selection) {
		if strings.TrimSpace(h6.Text()) != "Project links" {
			return
		}
		h6.NextFiltered("ul.vertical-tabs__list").Find("li a").Each(func(_ int, a *goquery.Selection) {
			href, _ := a.Attr("href")
			links = append(links, ProjectLink{
				PackageName: packageName,
				Version:     version,
				Label:       strings.TrimSpace(a.Text()),
				URL:         href,
				Verified:    verified,
			})
		})
	})
	return links
}

func parseProjectLinks(doc *goquery.Document, packageName, version string) []ProjectLink {
	var links []ProjectLink
	links = append(links, parseProjectLinksIn(verifiedSection(doc), true, packageName, version)...)
	links = append(links, parseProjectLinksIn(unverifiedSection(doc), false, packageName, version)...)
	return links
}

// Classifier: entity master, dedup di seluruh run oleh writer.
type Classifier struct {
	Category string `json:"category"`
	Value    string `json:"value"`
}

type TaggedWith struct {
	PackageName string `json:"package_name"`
	Version     string `json:"version"`
	Category    string `json:"category"`
	Value       string `json:"value"`
}

func parseClassifiers(doc *goquery.Document, packageName, version string) ([]Classifier, []TaggedWith) {
	var classifiers []Classifier
	var tagged []TaggedWith

	doc.Find(".sidebar-section__classifiers").First().ChildrenFiltered("li").Each(func(_ int, li *goquery.Selection) {
		category := strings.TrimSpace(li.Find("strong").First().Text())
		li.Find("ul li a").Each(func(_ int, a *goquery.Selection) {
			value := strings.TrimSpace(a.Text())
			classifiers = append(classifiers, Classifier{Category: category, Value: value})
			tagged = append(tagged, TaggedWith{
				PackageName: packageName,
				Version:     version,
				Category:    category,
				Value:       value,
			})
		})
	})

	return classifiers, tagged
}

type Maintainer struct {
	Username string `json:"username"`
	JoinedAt string `json:"joined_at"`
}

type MaintainedBy struct {
	PackageName        string `json:"package_name"`
	MaintainerUsername string `json:"maintainer_username"`
}

func parseMaintainerUsernames(doc *goquery.Document, packageName string) ([]string, []MaintainedBy) {
	var usernames []string
	var maintainedBy []MaintainedBy

	section := verifiedSection(doc)
	hasMaintainers := false
	section.Find("h6").Each(func(_ int, h6 *goquery.Selection) {
		if strings.TrimSpace(h6.Text()) == "Maintainers" {
			hasMaintainers = true
		}
	})
	if hasMaintainers {
		section.Find(".sidebar-section__user-gravatar-text").Each(func(_ int, s *goquery.Selection) {
			username := strings.TrimSpace(s.Text())
			usernames = append(usernames, username)
			maintainedBy = append(maintainedBy, MaintainedBy{
				PackageName:        packageName,
				MaintainerUsername: username,
			})
		})
	}

	return usernames, maintainedBy
}

// parseMaintainerJoinedAt: sebagian akun (mis. milik organisasi) tidak render metadiv "Date joined" sama sekali — itu bukan error, hasilnya cuma "".
func parseMaintainerJoinedAt(doc *goquery.Document) string {
	var joinedAt string
	doc.Find(".author-profile__metadiv").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		if s.Find("i.fa-calendar-alt").Length() == 0 {
			return true
		}
		joinedAt, _ = s.Find("time").First().Attr("datetime")
		return false
	})
	return joinedAt
}

func ParseMaintainerProfile(html []byte) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(html)))
	if err != nil {
		return "", fmt.Errorf("parsing HTML profil gagal: %w", err)
	}
	return parseMaintainerJoinedAt(doc), nil
}

// Page: gabungan semua entity dari satu fetch halaman project (utama atau per-versi).
type Page struct {
	Package        Package
	Organization   *Organization
	Release        Release
	ReleaseDetail  ReleaseDetail
	Files          []ReleaseFile
	Hashes         []FileHash
	FileTags       []ReleaseFileTag
	Attestations   []Attestation
	ProjectLinks   []ProjectLink
	Classifiers    []Classifier
	Tagged         []TaggedWith
	Maintainers    []string
	MaintainedBy   []MaintainedBy
	ReleaseHistory []ReleaseHistoryEntry
}

// CurrentVersion: halaman utama tanpa versi menampilkan rilis STABLE terbaru, bukan entry terbaru di release-history yang juga memuat pre-release (contoh nyata: pydantic 2.13.4 stable vs 2.14.0a1 pre-release).
func CurrentVersion(html []byte) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(html)))
	if err != nil {
		return "", fmt.Errorf("parsing HTML gagal: %w", err)
	}
	text := strings.TrimSpace(doc.Find(".package-header__name").First().Text())
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", nil
	}
	return fields[len(fields)-1], nil
}

// ParsePage: version sudah diketahui caller, tidak perlu diturunkan ulang dari teks header.
func ParsePage(html []byte, packageName, version string) (Page, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(html)))
	if err != nil {
		return Page{}, fmt.Errorf("parsing HTML untuk %s gagal: %w", packageName, err)
	}

	pkg, org := parsePackageAndOrg(doc)
	pkg.Name = packageName

	release, detail := parseReleaseMeta(doc, packageName, version)
	files, hashes, tags, attestations := parseReleaseFiles(doc, packageName, version)
	links := parseProjectLinks(doc, packageName, version)
	classifiers, tagged := parseClassifiers(doc, packageName, version)
	maintainers, maintainedBy := parseMaintainerUsernames(doc, packageName)
	history := parseReleaseHistory(doc)

	for _, entry := range history {
		if entry.Version == version {
			release.IsPrerelease = entry.IsPrerelease
			release.Yanked = entry.Yanked
			release.YankedReason = entry.YankedReason
			break
		}
	}

	return Page{
		Package:        pkg,
		Organization:   org,
		Release:        release,
		ReleaseDetail:  detail,
		Files:          files,
		Hashes:         hashes,
		FileTags:       tags,
		Attestations:   attestations,
		ProjectLinks:   links,
		Classifiers:    classifiers,
		Tagged:         tagged,
		Maintainers:    maintainers,
		MaintainedBy:   maintainedBy,
		ReleaseHistory: history,
	}, nil
}
