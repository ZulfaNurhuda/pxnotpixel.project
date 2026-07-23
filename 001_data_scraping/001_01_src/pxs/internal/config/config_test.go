package config

import "testing"

func TestConfigValues(t *testing.T) {
	if NumPackages != 100 {
		t.Fatalf("expected NumPackages=100, got %d", NumPackages)
	}
	if MaxReleasesPerPackage != 10 {
		t.Fatalf("expected MaxReleasesPerPackage=10, got %d", MaxReleasesPerPackage)
	}
	if InterPackageDelay.Seconds() != 5 {
		t.Fatalf("expected InterPackageDelay=5s, got %v", InterPackageDelay)
	}
	if UserAgentVersion == "" {
		t.Fatal("expected UserAgentVersion to be set")
	}
	if PyPIBaseURL != "https://pypi.org" {
		t.Fatalf("expected PyPIBaseURL='https://pypi.org', got %q", PyPIBaseURL)
	}
	if OutputTimestampFormat != "20060102T150405" {
		t.Fatalf("expected OutputTimestampFormat to be ISO 8601 basic format, got %q", OutputTimestampFormat)
	}
}
