-- Data Definition Language SQL untuk PyPI Extraction Data
-- sumber: data_modelling/relational/px_RELATIONAL_DETAILS.md

-- Script bersifat IDEMPOTEN, aman dijalankan berkali-kali.

-- buat database kalau belum ada
SELECT 'CREATE DATABASE px_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'px_db')\gexec

\c px_db

-- collation case-insensitive
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_collation WHERE collname = 'case_insensitive'
    ) THEN
        CREATE COLLATION case_insensitive (
            provider = icu, locale = 'und-u-ks-level2', deterministic = false
        );
    END IF;
END$$;

-- uuidv7() native di PG18+, fallback pure-SQL untuk versi dibawahnya
DO $outer$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_proc WHERE proname = 'uuidv7' AND pronargs = 0
    ) THEN
        EXECUTE $sql$
            CREATE FUNCTION uuidv7() RETURNS UUID AS $body$
                SELECT encode(
                  set_bit(
                    set_bit(
                      overlay(uuid_send(gen_random_uuid()) placing
                        substring(int8send((extract(epoch from clock_timestamp())*1000)::BIGINT) from 3)
                        from 1 for 6),
                      52, 1),
                    53, 1), 'hex')::UUID;
            $body$ LANGUAGE sql VOLATILE;
        $sql$;
    END IF;
END
$outer$;

-- enum type
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'LIFECYCLE_STATUS_ENUM') THEN
        CREATE TYPE LIFECYCLE_STATUS_ENUM AS ENUM ('archived', 'deprecated', 'quarantined');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'HASH_ALGORITHM_ENUM') THEN
        CREATE TYPE HASH_ALGORITHM_ENUM AS ENUM ('SHA256', 'MD5', 'BLAKE2b-256');
    END IF;
END$$;

-- tabel, urut sesuai dependency

CREATE TABLE IF NOT EXISTS organization (
    organization_id   UUID           NOT NULL DEFAULT uuidv7(),
    display_name      VARCHAR(255)   NOT NULL,
    name              TEXT           NOT NULL,
    CONSTRAINT pk_organization PRIMARY KEY (organization_id),
    CONSTRAINT uq_organization_name UNIQUE (name)
);

CREATE TABLE IF NOT EXISTS maintainer (
    maintainer_id   UUID                            NOT NULL DEFAULT uuidv7(),
    joined_at       TIMESTAMPTZ,
    username        TEXT COLLATE case_insensitive   NOT NULL,
    CONSTRAINT pk_maintainer PRIMARY KEY (maintainer_id),
    CONSTRAINT uq_maintainer_username UNIQUE (username),
    CONSTRAINT ck_maintainer_username_length CHECK (length(username) <= 50)
);

CREATE TABLE IF NOT EXISTS classifier (
    classifier_id   INTEGER        GENERATED ALWAYS AS IDENTITY,
    category        VARCHAR(50)    NOT NULL,
    value           VARCHAR(255)   NOT NULL,
    CONSTRAINT pk_classifier PRIMARY KEY (classifier_id),
    CONSTRAINT uq_classifier_category_value UNIQUE (category, value)
);

CREATE TABLE IF NOT EXISTS package (
    package_id         UUID                     NOT NULL DEFAULT uuidv7(),
    org_id             UUID,
    lifecycle_status   LIFECYCLE_STATUS_ENUM,
    name               TEXT                     NOT NULL,
    CONSTRAINT pk_package PRIMARY KEY (package_id),
    CONSTRAINT uq_package_name UNIQUE (name),
    CONSTRAINT fk_package_org_id_organization FOREIGN KEY (org_id) REFERENCES organization (organization_id)
);

CREATE TABLE IF NOT EXISTS release (
    release_id         UUID                     NOT NULL DEFAULT uuidv7(),
    package_id         UUID                     NOT NULL,
    created            TIMESTAMPTZ              NOT NULL,
    is_prerelease      BOOLEAN                  NOT NULL DEFAULT false,
    yanked             BOOLEAN                  NOT NULL DEFAULT false,
    lifecycle_status   LIFECYCLE_STATUS_ENUM,
    version            TEXT                     NOT NULL,
    yanked_reason      TEXT,
    summary            VARCHAR(512),
    license            TEXT,
    requires_python    TEXT,
    CONSTRAINT pk_release PRIMARY KEY (release_id),
    CONSTRAINT uq_release_package_id_version UNIQUE (package_id, version),
    CONSTRAINT fk_release_package_id_package FOREIGN KEY (package_id) REFERENCES package (package_id)
);

CREATE TABLE IF NOT EXISTS release_detail (
    release_id                       UUID      NOT NULL,
    meta_author_email_verified       BOOLEAN   DEFAULT false,
    meta_maintainer_email_verified   BOOLEAN   DEFAULT false,
    description                      TEXT,
    meta_author                      TEXT,
    meta_author_email                TEXT COLLATE case_insensitive,
    meta_maintainer                  TEXT,
    meta_maintainer_email            TEXT COLLATE case_insensitive,
    CONSTRAINT pk_release_detail PRIMARY KEY (release_id),
    CONSTRAINT fk_release_detail_release_id_release FOREIGN KEY (release_id) REFERENCES release (release_id)
);

CREATE TABLE IF NOT EXISTS release_file (
    release_file_id         UUID            NOT NULL DEFAULT uuidv7(),
    release_id              UUID            NOT NULL,
    size                    BIGINT          NOT NULL,
    upload_time             TIMESTAMPTZ     NOT NULL,
    is_trusted_publishing   BOOLEAN         NOT NULL DEFAULT false,
    filename                TEXT            NOT NULL,
    path                    TEXT            NOT NULL,
    uploaded_via            VARCHAR(255),
    CONSTRAINT pk_release_file PRIMARY KEY (release_file_id),
    CONSTRAINT uq_release_file_release_id_filename UNIQUE (release_id, filename),
    CONSTRAINT fk_release_file_release_id_release FOREIGN KEY (release_id) REFERENCES release (release_id)
);

CREATE TABLE IF NOT EXISTS file_hash (
    release_file_id   UUID                  NOT NULL,
    algorithm         HASH_ALGORITHM_ENUM   NOT NULL,
    digest            VARCHAR(64)           NOT NULL,
    CONSTRAINT pk_file_hash PRIMARY KEY (release_file_id, algorithm),
    CONSTRAINT fk_file_hash_release_file_id_release_file FOREIGN KEY (release_file_id) REFERENCES release_file (release_file_id)
);

CREATE TABLE IF NOT EXISTS project_link (
    release_id   UUID                            NOT NULL,
    label        TEXT COLLATE case_insensitive   NOT NULL,
    verified     BOOLEAN                         NOT NULL DEFAULT false,
    url          TEXT                            NOT NULL,
    CONSTRAINT pk_project_link PRIMARY KEY (release_id, label),
    CONSTRAINT fk_project_link_release_id_release FOREIGN KEY (release_id) REFERENCES release (release_id)
);

CREATE TABLE IF NOT EXISTS maintained_by (
    package_id      UUID   NOT NULL,
    maintainer_id   UUID   NOT NULL,
    CONSTRAINT pk_maintained_by PRIMARY KEY (package_id, maintainer_id),
    CONSTRAINT fk_maintained_by_package_id_package FOREIGN KEY (package_id) REFERENCES package (package_id),
    CONSTRAINT fk_maintained_by_maintainer_id_maintainer FOREIGN KEY (maintainer_id) REFERENCES maintainer (maintainer_id)
);

CREATE TABLE IF NOT EXISTS tagged_with (
    release_id      UUID      NOT NULL,
    classifier_id   INTEGER   NOT NULL,
    CONSTRAINT pk_tagged_with PRIMARY KEY (release_id, classifier_id),
    CONSTRAINT fk_tagged_with_release_id_release FOREIGN KEY (release_id) REFERENCES release (release_id),
    CONSTRAINT fk_tagged_with_classifier_id_classifier FOREIGN KEY (classifier_id) REFERENCES classifier (classifier_id)
);

CREATE TABLE IF NOT EXISTS release_keyword (
    release_id   UUID           NOT NULL,
    keyword      VARCHAR(100)   NOT NULL,
    CONSTRAINT pk_release_keyword PRIMARY KEY (release_id, keyword),
    CONSTRAINT fk_release_keyword_release_id_release FOREIGN KEY (release_id) REFERENCES release (release_id)
);

CREATE TABLE IF NOT EXISTS release_extra (
    release_id   UUID           NOT NULL,
    extra_name   VARCHAR(100)   NOT NULL,
    CONSTRAINT pk_release_extra PRIMARY KEY (release_id, extra_name),
    CONSTRAINT fk_release_extra_release_id_release FOREIGN KEY (release_id) REFERENCES release (release_id)
);

CREATE TABLE IF NOT EXISTS release_file_tag (
    release_file_id   UUID           NOT NULL,
    wheel_tag         VARCHAR(100)   NOT NULL,
    CONSTRAINT pk_release_file_tag PRIMARY KEY (release_file_id, wheel_tag),
    CONSTRAINT fk_release_file_tag_release_file_id_release_file FOREIGN KEY (release_file_id) REFERENCES release_file (release_file_id)
);

CREATE TABLE IF NOT EXISTS attestation (
    sigstore_log_index   BIGINT         NOT NULL,
    release_file_id      UUID           NOT NULL,
    integration_time     TIMESTAMPTZ    NOT NULL,
    statement_type       VARCHAR(255)   NOT NULL,
    predicate_type       VARCHAR(255)   NOT NULL,
    subject_name         VARCHAR(255)   NOT NULL,
    subject_digest       VARCHAR(64)    NOT NULL,
    source_repo          TEXT COLLATE case_insensitive,
    source_reference     VARCHAR(255),
    token_issuer         VARCHAR(255)   NOT NULL,
    runner_environment   VARCHAR(50),
    publisher_workflow   VARCHAR(255),
    trigger_event        VARCHAR(50),
    CONSTRAINT pk_attestation PRIMARY KEY (sigstore_log_index),
    CONSTRAINT fk_attestation_release_file_id_release_file FOREIGN KEY (release_file_id) REFERENCES release_file (release_file_id)
);
