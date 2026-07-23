package load

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"pxdl/internal/entity"
)

// Pakai time.RFC3339 karena time.Parse mengizinkan komponen detik-pecahan opsional khusus untuk layout ini.
// isoNoColonOffset matches pypi.org's actual datetime attribute format
// (e.g. "2014-10-13T07:59:07+0000") — a numeric UTC offset without the
// colon RFC3339 requires ("+00:00"), still valid ISO 8601.
const isoNoColonOffset = "2006-01-02T15:04:05Z0700"

func parseTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	t, err := time.Parse(isoNoColonOffset, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("format tanggal tidak valid %q: %w", s, err)
	}
	return t, nil
}

func parseNullableTime(s *string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	t, err := parseTime(*s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// CASCADE membuat urutan truncate tidak relevan terlepas arah FK.
func TruncateAll(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `TRUNCATE TABLE
		organization, maintainer, classifier, package, release, release_detail,
		release_file, file_hash, project_link, maintained_by, tagged_with,
		release_keyword, release_extra, release_file_tag, attestation
		CASCADE`)
	if err != nil {
		return fmt.Errorf("truncate gagal: %w", err)
	}
	return nil
}

func Organizations(ctx context.Context, tx pgx.Tx, rows []entity.Organization, r *Resolver) (int, error) {
	for _, o := range rows {
		var id string
		err := tx.QueryRow(ctx,
			`INSERT INTO organization (display_name, name) VALUES ($1, $2) RETURNING organization_id::text`,
			o.DisplayName, o.Name,
		).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("insert organization %q gagal: %w", o.Name, err)
		}
		r.orgID[o.Name] = id
	}
	return len(rows), nil
}

func Maintainers(ctx context.Context, tx pgx.Tx, rows []entity.Maintainer, r *Resolver) (int, error) {
	for _, m := range rows {
		joinedAt, err := parseNullableTime(m.JoinedAt)
		if err != nil {
			return 0, fmt.Errorf("insert maintainer %q gagal: %w", m.Username, err)
		}
		var id string
		err = tx.QueryRow(ctx,
			`INSERT INTO maintainer (joined_at, username) VALUES ($1, $2) RETURNING maintainer_id::text`,
			joinedAt, m.Username,
		).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("insert maintainer %q gagal: %w", m.Username, err)
		}
		r.maintainerID[m.Username] = id
	}
	return len(rows), nil
}

func Classifiers(ctx context.Context, tx pgx.Tx, rows []entity.Classifier, r *Resolver) (int, error) {
	for _, c := range rows {
		var id string
		err := tx.QueryRow(ctx,
			`INSERT INTO classifier (category, value) VALUES ($1, $2) RETURNING classifier_id::text`,
			c.Category, c.Value,
		).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("insert classifier %s=%s gagal: %w", c.Category, c.Value, err)
		}
		r.classifierID[classifierKey{Category: c.Category, Value: c.Value}] = id
	}
	return len(rows), nil
}

func Packages(ctx context.Context, tx pgx.Tx, rows []entity.Package, r *Resolver) (int, error) {
	for _, p := range rows {
		orgName := ""
		if p.OrganizationOwner != nil {
			orgName = *p.OrganizationOwner
		}
		orgID, err := r.lookupOrg(orgName)
		if err != nil {
			return 0, fmt.Errorf("insert package %q gagal: %w", p.Name, err)
		}
		var id string
		err = tx.QueryRow(ctx,
			`INSERT INTO package (org_id, lifecycle_status, name) VALUES ($1::uuid, $2, $3) RETURNING package_id::text`,
			orgID, p.LifecycleStatus, p.Name,
		).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("insert package %q gagal: %w", p.Name, err)
		}
		r.packageID[p.Name] = id
	}
	return len(rows), nil
}

func Releases(ctx context.Context, tx pgx.Tx, rows []entity.Release, r *Resolver) (int, error) {
	for _, rel := range rows {
		packageID, err := r.lookupPackage(rel.PackageName)
		if err != nil {
			return 0, fmt.Errorf("insert release %s %s gagal: %w", rel.PackageName, rel.Version, err)
		}
		created, err := parseTime(rel.Created)
		if err != nil {
			return 0, fmt.Errorf("insert release %s %s gagal: %w", rel.PackageName, rel.Version, err)
		}
		var id string
		err = tx.QueryRow(ctx,
			`INSERT INTO release (package_id, created, is_prerelease, yanked, lifecycle_status, version, yanked_reason, summary, license, requires_python)
			 VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING release_id::text`,
			packageID, created, rel.IsPrerelease, rel.Yanked, rel.LifecycleStatus, rel.Version, rel.YankedReason, rel.Summary, rel.License, rel.RequiresPython,
		).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("insert release %s %s gagal: %w", rel.PackageName, rel.Version, err)
		}
		r.releaseID[releaseKey{PackageName: rel.PackageName, Version: rel.Version}] = id
	}
	return len(rows), nil
}

func ReleaseDetails(ctx context.Context, tx pgx.Tx, rows []entity.ReleaseDetail, r *Resolver) (int, error) {
	for _, d := range rows {
		releaseID, err := r.lookupRelease(d.PackageName, d.Version)
		if err != nil {
			return 0, fmt.Errorf("insert release_detail %s %s gagal: %w", d.PackageName, d.Version, err)
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO release_detail (release_id, meta_author_email_verified, meta_maintainer_email_verified, description, meta_author, meta_author_email, meta_maintainer, meta_maintainer_email)
			 VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8)`,
			releaseID, d.MetaAuthorEmailVerified, d.MetaMaintainerEmailVerified, d.Description, d.MetaAuthor, d.MetaAuthorEmail, d.MetaMaintainer, d.MetaMaintainerEmail,
		)
		if err != nil {
			return 0, fmt.Errorf("insert release_detail %s %s gagal: %w", d.PackageName, d.Version, err)
		}
	}
	return len(rows), nil
}

func ReleaseFiles(ctx context.Context, tx pgx.Tx, rows []entity.ReleaseFile, r *Resolver) (int, error) {
	for _, f := range rows {
		releaseID, err := r.lookupRelease(f.PackageName, f.Version)
		if err != nil {
			return 0, fmt.Errorf("insert release_file %s gagal: %w", f.Filename, err)
		}
		uploadTime, err := parseTime(f.UploadTime)
		if err != nil {
			return 0, fmt.Errorf("insert release_file %s gagal: %w", f.Filename, err)
		}
		var id string
		err = tx.QueryRow(ctx,
			`INSERT INTO release_file (release_id, size, upload_time, is_trusted_publishing, filename, path, uploaded_via)
			 VALUES ($1::uuid, $2, $3, $4, $5, $6, $7) RETURNING release_file_id::text`,
			releaseID, f.Size, uploadTime, f.IsTrustedPublishing, f.Filename, f.Path, f.UploadedVia,
		).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("insert release_file %s gagal: %w", f.Filename, err)
		}
		r.fileID[fileKey{PackageName: f.PackageName, Version: f.Version, Filename: f.Filename}] = id
	}
	return len(rows), nil
}

func FileHashes(ctx context.Context, tx pgx.Tx, rows []entity.FileHash, r *Resolver) (int, error) {
	for _, h := range rows {
		fileID, err := r.lookupFile(h.PackageName, h.Version, h.Filename)
		if err != nil {
			return 0, fmt.Errorf("insert file_hash %s gagal: %w", h.Filename, err)
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO file_hash (release_file_id, algorithm, digest) VALUES ($1::uuid, $2, $3)`,
			fileID, h.Algorithm, h.Digest,
		)
		if err != nil {
			return 0, fmt.Errorf("insert file_hash %s gagal: %w", h.Filename, err)
		}
	}
	return len(rows), nil
}

func ProjectLinks(ctx context.Context, tx pgx.Tx, rows []entity.ProjectLink, r *Resolver) (int, error) {
	for _, l := range rows {
		releaseID, err := r.lookupRelease(l.PackageName, l.Version)
		if err != nil {
			return 0, fmt.Errorf("insert project_link %s gagal: %w", l.Label, err)
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO project_link (release_id, label, verified, url) VALUES ($1::uuid, $2, $3, $4)`,
			releaseID, l.Label, l.Verified, l.URL,
		)
		if err != nil {
			return 0, fmt.Errorf("insert project_link %s gagal: %w", l.Label, err)
		}
	}
	return len(rows), nil
}

func MaintainedBy(ctx context.Context, tx pgx.Tx, rows []entity.MaintainedBy, r *Resolver) (int, error) {
	for _, mb := range rows {
		packageID, err := r.lookupPackage(mb.PackageName)
		if err != nil {
			return 0, fmt.Errorf("insert maintained_by %s/%s gagal: %w", mb.PackageName, mb.MaintainerUsername, err)
		}
		maintainerID, err := r.lookupMaintainer(mb.MaintainerUsername)
		if err != nil {
			return 0, fmt.Errorf("insert maintained_by %s/%s gagal: %w", mb.PackageName, mb.MaintainerUsername, err)
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO maintained_by (package_id, maintainer_id) VALUES ($1::uuid, $2::uuid)`,
			packageID, maintainerID,
		)
		if err != nil {
			return 0, fmt.Errorf("insert maintained_by %s/%s gagal: %w", mb.PackageName, mb.MaintainerUsername, err)
		}
	}
	return len(rows), nil
}

func TaggedWith(ctx context.Context, tx pgx.Tx, rows []entity.TaggedWith, r *Resolver) (int, error) {
	for _, t := range rows {
		releaseID, err := r.lookupRelease(t.PackageName, t.Version)
		if err != nil {
			return 0, fmt.Errorf("insert tagged_with %s %s gagal: %w", t.PackageName, t.Version, err)
		}
		classifierID, err := r.lookupClassifier(t.Category, t.Value)
		if err != nil {
			return 0, fmt.Errorf("insert tagged_with %s %s gagal: %w", t.PackageName, t.Version, err)
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO tagged_with (release_id, classifier_id) VALUES ($1::uuid, $2::int)`,
			releaseID, classifierID,
		)
		if err != nil {
			return 0, fmt.Errorf("insert tagged_with %s %s gagal: %w", t.PackageName, t.Version, err)
		}
	}
	return len(rows), nil
}

func ReleaseKeywords(ctx context.Context, tx pgx.Tx, rows []entity.ReleaseKeyword, r *Resolver) (int, error) {
	for _, k := range rows {
		releaseID, err := r.lookupRelease(k.PackageName, k.Version)
		if err != nil {
			return 0, fmt.Errorf("insert release_keyword %s gagal: %w", k.Keyword, err)
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO release_keyword (release_id, keyword) VALUES ($1::uuid, $2)`,
			releaseID, k.Keyword,
		)
		if err != nil {
			return 0, fmt.Errorf("insert release_keyword %s gagal: %w", k.Keyword, err)
		}
	}
	return len(rows), nil
}

func ReleaseExtras(ctx context.Context, tx pgx.Tx, rows []entity.ReleaseExtra, r *Resolver) (int, error) {
	for _, e := range rows {
		releaseID, err := r.lookupRelease(e.PackageName, e.Version)
		if err != nil {
			return 0, fmt.Errorf("insert release_extra %s gagal: %w", e.ExtraName, err)
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO release_extra (release_id, extra_name) VALUES ($1::uuid, $2)`,
			releaseID, e.ExtraName,
		)
		if err != nil {
			return 0, fmt.Errorf("insert release_extra %s gagal: %w", e.ExtraName, err)
		}
	}
	return len(rows), nil
}

func ReleaseFileTags(ctx context.Context, tx pgx.Tx, rows []entity.ReleaseFileTag, r *Resolver) (int, error) {
	for _, t := range rows {
		fileID, err := r.lookupFile(t.PackageName, t.Version, t.Filename)
		if err != nil {
			return 0, fmt.Errorf("insert release_file_tag %s gagal: %w", t.WheelTag, err)
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO release_file_tag (release_file_id, wheel_tag) VALUES ($1::uuid, $2)`,
			fileID, t.WheelTag,
		)
		if err != nil {
			return 0, fmt.Errorf("insert release_file_tag %s gagal: %w", t.WheelTag, err)
		}
	}
	return len(rows), nil
}

func Attestations(ctx context.Context, tx pgx.Tx, rows []entity.Attestation, r *Resolver) (int, error) {
	for _, a := range rows {
		fileID, err := r.lookupFile(a.PackageName, a.Version, a.Filename)
		if err != nil {
			return 0, fmt.Errorf("insert attestation (sigstore_log_index=%d) gagal: %w", a.SigstoreLogIndex, err)
		}
		integrationTime, err := parseTime(a.IntegrationTime)
		if err != nil {
			return 0, fmt.Errorf("insert attestation (sigstore_log_index=%d) gagal: %w", a.SigstoreLogIndex, err)
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO attestation (sigstore_log_index, release_file_id, integration_time, statement_type, predicate_type, subject_name, subject_digest, source_repo, source_reference, token_issuer, runner_environment, publisher_workflow, trigger_event)
			 VALUES ($1, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
			a.SigstoreLogIndex, fileID, integrationTime, a.StatementType, a.PredicateType, a.SubjectName, a.SubjectDigest, a.SourceRepo, a.SourceReference, a.TokenIssuer, a.RunnerEnvironment, a.PublisherWorkflow, a.TriggerEvent,
		)
		if err != nil {
			return 0, fmt.Errorf("insert attestation (sigstore_log_index=%d) gagal: %w", a.SigstoreLogIndex, err)
		}
	}
	return len(rows), nil
}
