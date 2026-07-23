from clean import CleaningReport, clean_entity


def test_missing_optional_column_filled_with_none(capsys):
    report = CleaningReport()
    rows = [{"name": "boto3"}]  # lifecycle_status, organization_owner absen
    result = clean_entity("package.json", rows, report)
    assert result == [{"name": "boto3", "lifecycle_status": None, "organization_owner": None}]
    assert report.dropped_count == 0


def test_missing_required_column_drops_row(capsys):
    report = CleaningReport()
    rows = [{"lifecycle_status": None}]  # "name" wajib, absen
    result = clean_entity("package.json", rows, report)
    assert result == []
    assert report.dropped_count == 1
    captured = capsys.readouterr()
    assert "Lewati baris package.json" in captured.out
    assert "name kosong tapi wajib diisi" in captured.out


def test_identifier_over_length_drops_row():
    report = CleaningReport()
    rows = [{"username": "x" * 51, "joined_at": None}]
    result = clean_entity("maintainer.json", rows, report)
    assert result == []
    assert report.dropped_count == 1


def test_descriptive_over_length_is_truncated_with_ellipsis():
    report = CleaningReport()
    rows = [{"name": "org1", "display_name": "x" * 260}]
    result = clean_entity("organization.json", rows, report)
    assert len(result) == 1
    assert result[0]["display_name"] == "x" * 252 + "..."
    assert len(result[0]["display_name"]) == 255
    assert report.truncated_count == 1
    assert report.dropped_count == 0


def test_invalid_enum_value_drops_row():
    report = CleaningReport()
    rows = [{"name": "pkg", "lifecycle_status": "not-a-real-status", "organization_owner": None}]
    result = clean_entity("package.json", rows, report)
    assert result == []
    assert report.dropped_count == 1


def test_valid_row_passes_through_unchanged():
    report = CleaningReport()
    rows = [{"name": "pkg", "lifecycle_status": "archived", "organization_owner": "someorg"}]
    result = clean_entity("package.json", rows, report)
    assert result == [{"name": "pkg", "lifecycle_status": "archived", "organization_owner": "someorg"}]
    assert report.dropped_count == 0
    assert report.truncated_count == 0


def test_empty_string_treated_same_as_missing():
    report = CleaningReport()
    rows = [{"name": "pkg", "lifecycle_status": "", "organization_owner": ""}]
    result = clean_entity("package.json", rows, report)
    assert result == [{"name": "pkg", "lifecycle_status": None, "organization_owner": None}]


from clean import backfill_maintainers


def test_backfill_adds_placeholder_for_orphaned_maintainer():
    report = CleaningReport()
    maintainer_rows = [{"username": "alice", "joined_at": "2020-01-02T00:00:00+0000"}]
    maintained_by_rows = [
        {"package_name": "pkg1", "maintainer_username": "alice"},
        {"package_name": "pkg2", "maintainer_username": "bob"},  # bob tak punya baris maintainer
    ]
    result = backfill_maintainers(maintainer_rows, maintained_by_rows, report)
    assert {"username": "alice", "joined_at": "2020-01-02T00:00:00+0000"} in result
    assert {"username": "bob", "joined_at": None} in result
    assert len(result) == 2
    assert report.backfilled_count == 1


def test_backfill_does_not_duplicate_known_maintainer():
    report = CleaningReport()
    maintainer_rows = [{"username": "alice", "joined_at": None}]
    maintained_by_rows = [{"package_name": "pkg1", "maintainer_username": "alice"}]
    result = backfill_maintainers(maintainer_rows, maintained_by_rows, report)
    assert result == [{"username": "alice", "joined_at": None}]
    assert report.backfilled_count == 0


def test_backfill_adds_one_placeholder_even_if_referenced_multiple_times():
    report = CleaningReport()
    maintainer_rows = []
    maintained_by_rows = [
        {"package_name": "pkg1", "maintainer_username": "carol"},
        {"package_name": "pkg2", "maintainer_username": "carol"},
    ]
    result = backfill_maintainers(maintainer_rows, maintained_by_rows, report)
    assert result == [{"username": "carol", "joined_at": None}]
    assert report.backfilled_count == 1


def test_maintained_by_overlong_username_dropped_not_resurrected():
    # baris maintainer.json kepanjangan di-drop oleh clean_entity
    maintainer_report = CleaningReport()
    overlong = "x" * 51
    maintainer_rows = clean_entity(
        "maintainer.json", [{"username": overlong, "joined_at": None}], maintainer_report
    )
    assert maintainer_rows == []
    assert maintainer_report.dropped_count == 1

    # baris maintained_by.json yang cocok juga harus di-drop, agar backfill tak menghidupkannya kembali sebagai placeholder
    mb_report = CleaningReport()
    maintained_by_rows = clean_entity(
        "maintained_by.json",
        [{"package_name": "pkg1", "maintainer_username": overlong}],
        mb_report,
    )
    assert maintained_by_rows == []
    assert mb_report.dropped_count == 1

    backfill_report = CleaningReport()
    result = backfill_maintainers(maintainer_rows, maintained_by_rows, backfill_report)
    assert result == []
    assert backfill_report.backfilled_count == 0


def test_backfill_matches_maintainer_case_insensitively():
    report = CleaningReport()
    maintainer_rows = [{"username": "bob", "joined_at": "2020-01-02T00:00:00+0000"}]
    maintained_by_rows = [{"package_name": "pkg1", "maintainer_username": "Bob"}]
    result = backfill_maintainers(maintainer_rows, maintained_by_rows, report)
    assert result == [{"username": "bob", "joined_at": "2020-01-02T00:00:00+0000"}]
    assert report.backfilled_count == 0
