package entity

type Organization struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

type Maintainer struct {
	Username string  `json:"username"`
	JoinedAt *string `json:"joined_at"`
}

type Classifier struct {
	Category string `json:"category"`
	Value    string `json:"value"`
}

type Package struct {
	Name              string  `json:"name"`
	LifecycleStatus   *string `json:"lifecycle_status"`
	OrganizationOwner *string `json:"organization_owner"`
}

type Release struct {
	PackageName     string  `json:"package_name"`
	Version         string  `json:"version"`
	Created         string  `json:"created"`
	IsPrerelease    bool    `json:"is_prerelease"`
	Yanked          bool    `json:"yanked"`
	LifecycleStatus *string `json:"lifecycle_status"`
	YankedReason    *string `json:"yanked_reason"`
	Summary         *string `json:"summary"`
	License         *string `json:"license"`
	RequiresPython  *string `json:"requires_python"`
}

type ReleaseDetail struct {
	PackageName                string  `json:"package_name"`
	Version                    string  `json:"version"`
	Description                *string `json:"description"`
	MetaAuthor                 *string `json:"meta_author"`
	MetaAuthorEmail            *string `json:"meta_author_email"`
	MetaAuthorEmailVerified    *bool   `json:"meta_author_email_verified"`
	MetaMaintainer             *string `json:"meta_maintainer"`
	MetaMaintainerEmail        *string `json:"meta_maintainer_email"`
	MetaMaintainerEmailVerified *bool   `json:"meta_maintainer_email_verified"`
}

// JSON sumber juga punya "packagetype", sengaja tidak dipetakan karena tak ada kolom DB-nya.
type ReleaseFile struct {
	PackageName         string  `json:"package_name"`
	Version             string  `json:"version"`
	Filename            string  `json:"filename"`
	Path                string  `json:"path"`
	Size                int64   `json:"size"`
	UploadTime          string  `json:"upload_time"`
	IsTrustedPublishing bool    `json:"is_trusted_publishing"`
	UploadedVia         *string `json:"uploaded_via"`
}

type FileHash struct {
	PackageName string `json:"package_name"`
	Version     string `json:"version"`
	Filename    string `json:"filename"`
	Algorithm   string `json:"algorithm"`
	Digest      string `json:"digest"`
}

type ProjectLink struct {
	PackageName string `json:"package_name"`
	Version     string `json:"version"`
	Label       string `json:"label"`
	URL         string `json:"url"`
	Verified    bool   `json:"verified"`
}

type MaintainedBy struct {
	PackageName        string `json:"package_name"`
	MaintainerUsername string `json:"maintainer_username"`
}

type TaggedWith struct {
	PackageName string `json:"package_name"`
	Version     string `json:"version"`
	Category    string `json:"category"`
	Value       string `json:"value"`
}

type ReleaseKeyword struct {
	PackageName string `json:"package_name"`
	Version     string `json:"version"`
	Keyword     string `json:"keyword"`
}

type ReleaseExtra struct {
	PackageName string `json:"package_name"`
	Version     string `json:"version"`
	ExtraName   string `json:"extra_name"`
}

type ReleaseFileTag struct {
	PackageName string `json:"package_name"`
	Version     string `json:"version"`
	Filename    string `json:"filename"`
	WheelTag    string `json:"wheel_tag"`
}

type Attestation struct {
	PackageName       string  `json:"package_name"`
	Version           string  `json:"version"`
	Filename          string  `json:"filename"`
	SigstoreLogIndex  int64   `json:"sigstore_log_index"`
	IntegrationTime   string  `json:"integration_time"`
	StatementType     string  `json:"statement_type"`
	PredicateType     string  `json:"predicate_type"`
	SubjectName       string  `json:"subject_name"`
	SubjectDigest     string  `json:"subject_digest"`
	SourceRepo        *string `json:"source_repo"`
	SourceReference   *string `json:"source_reference"`
	TokenIssuer       string  `json:"token_issuer"`
	RunnerEnvironment *string `json:"runner_environment"`
	PublisherWorkflow *string `json:"publisher_workflow"`
	TriggerEvent      *string `json:"trigger_event"`
}
