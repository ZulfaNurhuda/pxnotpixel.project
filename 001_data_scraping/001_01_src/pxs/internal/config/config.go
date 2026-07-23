package config

import "time"

const (
	// NumPackages: jumlah package yang di-scrape per run.
	NumPackages = 100
	// MaxReleasesPerPackage: batas atas rilis historis per package (beberapa package punya ribuan rilis).
	MaxReleasesPerPackage = 10
	// InterPackageDelay: jeda minimum antar-package, TIDAK berlaku antar fetch versi/profil maintainer dalam package yang sama.
	InterPackageDelay = 5 * time.Second
	// HTTPTimeout: timeout per-request untuk fetch client.
	HTTPTimeout = 30 * time.Second
	// UserAgentVersion: versi pxs yang dilaporkan di header User-Agent.
	UserAgentVersion = "0.1.0"
	// PyPIBaseURL: base URL default untuk request ke pypi.org.
	PyPIBaseURL = "https://pypi.org"
	// OutputTimestampFormat: format ISO 8601 basic (tanpa "-"/":") agar aman jadi nama folder di semua OS.
	OutputTimestampFormat = "20060102T150405"
)
