from schema import SCHEMA, ENTITY_FILES


def test_all_15_entities_present():
    expected = {
        "package.json", "organization.json", "maintainer.json", "maintained_by.json",
        "release.json", "release_detail.json", "release_file.json", "file_hash.json",
        "project_link.json", "classifier.json", "tagged_with.json", "release_keyword.json",
        "release_extra.json", "release_file_tag.json", "attestation.json",
    }
    assert set(SCHEMA.keys()) == expected
    assert set(ENTITY_FILES) == expected


def test_maintainer_username_is_identifier_max_50():
    spec = SCHEMA["maintainer.json"]["username"]
    assert spec == {"nullable": False, "max_length": 50, "kind": "identifier", "enum": None}


def test_organization_display_name_is_descriptive_max_255():
    spec = SCHEMA["organization.json"]["display_name"]
    assert spec == {"nullable": False, "max_length": 255, "kind": "descriptive", "enum": None}


def test_release_lifecycle_status_is_nullable_enum():
    spec = SCHEMA["release.json"]["lifecycle_status"]
    assert spec["nullable"] is True
    assert spec["enum"] == ["archived", "deprecated", "quarantined"]


def test_file_hash_algorithm_enum():
    spec = SCHEMA["file_hash.json"]["algorithm"]
    assert spec["nullable"] is False
    assert spec["enum"] == ["SHA256", "MD5", "BLAKE2b-256"]


def test_maintained_by_maintainer_username_is_identifier_max_50():
    spec = SCHEMA["maintained_by.json"]["maintainer_username"]
    assert spec == {"nullable": False, "max_length": 50, "kind": "identifier", "enum": None}


def test_columns_with_no_length_limit_pass_through():
    # kolom TEXT di 01_px_INIT.sql tanpa cap VARCHAR, tak ada yang dipotong/di-drop
    assert SCHEMA["release.json"]["license"]["max_length"] is None
    assert SCHEMA["release_detail.json"]["description"]["max_length"] is None
    assert SCHEMA["release_file.json"]["packagetype"]["max_length"] is None
