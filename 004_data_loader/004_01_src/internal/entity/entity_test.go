package entity

import (
	"encoding/json"
	"testing"
)

func TestReleaseRoundTrip(t *testing.T) {
	input := `[{"package_name":"requests","version":"2.31.0","created":"2023-05-22T15:04:05Z","is_prerelease":false,"yanked":false,"lifecycle_status":null,"yanked_reason":null,"summary":"Python HTTP for Humans.","license":"Apache-2.0","requires_python":">=3.7"}]`

	var releases []Release
	if err := json.Unmarshal([]byte(input), &releases); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(releases) != 1 {
		t.Fatalf("got %d releases, want 1", len(releases))
	}
	r := releases[0]
	if r.PackageName != "requests" || r.Version != "2.31.0" || r.Created != "2023-05-22T15:04:05Z" {
		t.Fatalf("unexpected fields: %+v", r)
	}
	if r.IsPrerelease != false || r.Yanked != false {
		t.Fatalf("unexpected bool fields: %+v", r)
	}
	if r.LifecycleStatus != nil || r.YankedReason != nil {
		t.Fatalf("expected nil pointers for null fields: %+v", r)
	}
	if r.License == nil || *r.License != "Apache-2.0" {
		t.Fatalf("unexpected License: %+v", r)
	}
}

func TestReleaseFileIgnoresUnknownPackagetype(t *testing.T) {
	input := `[{"package_name":"requests","version":"2.31.0","filename":"requests-2.31.0-py3-none-any.whl","path":"a/b/requests-2.31.0-py3-none-any.whl","size":62574,"upload_time":"2023-05-22T15:04:05Z","is_trusted_publishing":true,"uploaded_via":"twine/4.0.2","packagetype":"bdist_wheel"}]`

	var files []ReleaseFile
	if err := json.Unmarshal([]byte(input), &files); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(files) != 1 || files[0].Filename != "requests-2.31.0-py3-none-any.whl" || files[0].Size != 62574 {
		t.Fatalf("unexpected result: %+v", files)
	}
}
