"""Registry kolom per entity, hasil encode px_RELATIONAL_DETAILS.md dan 01_px_INIT.sql sebagai data."""


def column(nullable=True, max_length=None, kind=None, enum=None):
    return {"nullable": nullable, "max_length": max_length, "kind": kind, "enum": enum}


_LIFECYCLE_STATUS_ENUM = ["archived", "deprecated", "quarantined"]
_HASH_ALGORITHM_ENUM = ["SHA256", "MD5", "BLAKE2b-256"]

SCHEMA = {
    "package.json": {
        "name": column(nullable=False),
        "lifecycle_status": column(nullable=True, enum=_LIFECYCLE_STATUS_ENUM),
        "organization_owner": column(nullable=True),
    },
    "organization.json": {
        "name": column(nullable=False),
        "display_name": column(nullable=False, max_length=255, kind="descriptive"),
    },
    "maintainer.json": {
        "username": column(nullable=False, max_length=50, kind="identifier"),
        "joined_at": column(nullable=True),
    },
    "maintained_by.json": {
        "package_name": column(nullable=False),
        "maintainer_username": column(nullable=False, max_length=50, kind="identifier"),
    },
    "release.json": {
        "package_name": column(nullable=False),
        "version": column(nullable=False),
        "created": column(nullable=False),
        "is_prerelease": column(nullable=False),
        "yanked": column(nullable=False),
        "lifecycle_status": column(nullable=True, enum=_LIFECYCLE_STATUS_ENUM),
        "yanked_reason": column(nullable=True),
        "summary": column(nullable=True, max_length=512, kind="descriptive"),
        "license": column(nullable=True),
        "requires_python": column(nullable=True),
    },
    "release_detail.json": {
        "package_name": column(nullable=False),
        "version": column(nullable=False),
        "description": column(nullable=True),
        "meta_author": column(nullable=True),
        "meta_author_email": column(nullable=True),
        "meta_author_email_verified": column(nullable=True),
        "meta_maintainer": column(nullable=True),
        "meta_maintainer_email": column(nullable=True),
        "meta_maintainer_email_verified": column(nullable=True),
    },
    "release_file.json": {
        "package_name": column(nullable=False),
        "version": column(nullable=False),
        "filename": column(nullable=False),
        "path": column(nullable=False),
        "size": column(nullable=False),
        "upload_time": column(nullable=False),
        "is_trusted_publishing": column(nullable=False),
        "uploaded_via": column(nullable=True, max_length=255, kind="descriptive"),
        "packagetype": column(nullable=True),
    },
    "file_hash.json": {
        "package_name": column(nullable=False),
        "version": column(nullable=False),
        "filename": column(nullable=False),
        "algorithm": column(nullable=False, enum=_HASH_ALGORITHM_ENUM),
        "digest": column(nullable=False, max_length=64, kind="identifier"),
    },
    "project_link.json": {
        "package_name": column(nullable=False),
        "version": column(nullable=False),
        "label": column(nullable=False),
        "url": column(nullable=False),
        "verified": column(nullable=False),
    },
    "classifier.json": {
        "category": column(nullable=False, max_length=50, kind="identifier"),
        "value": column(nullable=False, max_length=255, kind="identifier"),
    },
    "tagged_with.json": {
        "package_name": column(nullable=False),
        "version": column(nullable=False),
        "category": column(nullable=False),
        "value": column(nullable=False),
    },
    "release_keyword.json": {
        "package_name": column(nullable=False),
        "version": column(nullable=False),
        "keyword": column(nullable=False, max_length=100, kind="descriptive"),
    },
    "release_extra.json": {
        "package_name": column(nullable=False),
        "version": column(nullable=False),
        "extra_name": column(nullable=False, max_length=100, kind="identifier"),
    },
    "release_file_tag.json": {
        "package_name": column(nullable=False),
        "version": column(nullable=False),
        "filename": column(nullable=False),
        "wheel_tag": column(nullable=False, max_length=100, kind="descriptive"),
    },
    "attestation.json": {
        "package_name": column(nullable=False),
        "version": column(nullable=False),
        "filename": column(nullable=False),
        "sigstore_log_index": column(nullable=False),
        "integration_time": column(nullable=False),
        "statement_type": column(nullable=False, max_length=255, kind="identifier"),
        "predicate_type": column(nullable=False, max_length=255, kind="identifier"),
        "subject_name": column(nullable=False, max_length=255, kind="identifier"),
        "subject_digest": column(nullable=False, max_length=64, kind="identifier"),
        "source_repo": column(nullable=True),
        "source_reference": column(nullable=True, max_length=255, kind="identifier"),
        "token_issuer": column(nullable=False, max_length=255, kind="identifier"),
        "runner_environment": column(nullable=True, max_length=50, kind="identifier"),
        "publisher_workflow": column(nullable=True, max_length=255, kind="identifier"),
        "trigger_event": column(nullable=True, max_length=50, kind="identifier"),
    },
}

ENTITY_FILES = list(SCHEMA.keys())
