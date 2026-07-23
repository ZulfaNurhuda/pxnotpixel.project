from pathlib import Path

import pytest

from main import find_latest_run, read_entity, write_entity


def test_find_latest_run_picks_lexicographically_greatest(tmp_path):
    (tmp_path / "pxs_20260101T000000").mkdir()
    (tmp_path / "pxs_20260723T044432").mkdir()
    (tmp_path / "pxs_20260615T120000").mkdir()

    result = find_latest_run(tmp_path)

    assert result.name == "pxs_20260723T044432"


def test_find_latest_run_raises_when_none_exist(tmp_path):
    with pytest.raises(FileNotFoundError):
        find_latest_run(tmp_path)


def test_read_entity_fills_missing_keys_with_none(tmp_path):
    run_dir = tmp_path / "pxs_20260101T000000"
    run_dir.mkdir()
    (run_dir / "package.json").write_text(
        '[{"name":"boto3","lifecycle_status":"archived"},{"name":"packaging"}]',
        encoding="utf-8",
    )

    rows = read_entity(run_dir, "package.json")

    assert len(rows) == 2
    assert rows[0]["name"] == "boto3"
    assert rows[0]["lifecycle_status"] == "archived"
    assert rows[1]["name"] == "packaging"
    assert rows[1]["lifecycle_status"] is None


def test_write_entity_writes_valid_compact_json(tmp_path):
    out_dir = tmp_path / "out"
    out_dir.mkdir()
    rows = [{"name": "boto3", "lifecycle_status": None}]

    write_entity(out_dir, "package.json", rows)

    import json
    written = json.loads((out_dir / "package.json").read_text(encoding="utf-8"))
    assert written == rows


import json as json_module

from main import run


def _write_json(path, rows):
    path.write_text(json_module.dumps(rows), encoding="utf-8")


def test_run_end_to_end(tmp_path, capsys):
    data_root = tmp_path / "001_02_data"
    run_dir = data_root / "pxs_20260723T044432"
    run_dir.mkdir(parents=True)
    output_root = tmp_path / "003_02_data_cleaned"

    # package.json: 1 baris valid, 1 dengan lifecycle_status tidak valid (di-drop)
    _write_json(run_dir / "package.json", [
        {"name": "boto3", "lifecycle_status": "archived", "organization_owner": None},
        {"name": "badpkg", "lifecycle_status": "not-real", "organization_owner": None},
    ])
    _write_json(run_dir / "organization.json", [])
    # maintainer.json: hanya alice yang punya baris asli
    _write_json(run_dir / "maintainer.json", [
        {"username": "alice", "joined_at": "2020-01-02T00:00:00+0000"},
    ])
    # maintained_by.json: bob direferensikan tapi tak ada baris maintainer (di-backfill)
    _write_json(run_dir / "maintained_by.json", [
        {"package_name": "boto3", "maintainer_username": "alice"},
        {"package_name": "boto3", "maintainer_username": "bob"},
    ])
    _write_json(run_dir / "release.json", [
        {"package_name": "boto3", "version": "1.0.0", "created": "2026-01-02T00:00:00+0000",
         "is_prerelease": False, "yanked": False, "lifecycle_status": None,
         "yanked_reason": None, "summary": "x" * 520, "license": "Apache-2.0",
         "requires_python": ">=3.10"},
    ])
    _write_json(run_dir / "release_detail.json", [
        {"package_name": "boto3", "version": "1.0.0", "description": None,
         "meta_author": None, "meta_author_email": None, "meta_author_email_verified": False,
         "meta_maintainer": None, "meta_maintainer_email": None, "meta_maintainer_email_verified": False},
    ])
    _write_json(run_dir / "release_file.json", [])
    _write_json(run_dir / "file_hash.json", [])
    _write_json(run_dir / "project_link.json", [])
    _write_json(run_dir / "classifier.json", [])
    _write_json(run_dir / "tagged_with.json", [])
    _write_json(run_dir / "release_keyword.json", [])
    _write_json(run_dir / "release_extra.json", [])
    _write_json(run_dir / "release_file_tag.json", [])
    _write_json(run_dir / "attestation.json", [])

    exit_code = run(data_root, output_root)

    assert exit_code == 0

    out_dir = output_root / "pxs_20260723T044432"
    assert out_dir.is_dir()

    packages = json_module.loads((out_dir / "package.json").read_text(encoding="utf-8"))
    assert len(packages) == 1  # badpkg di-drop (lifecycle_status tidak valid)
    assert packages[0]["name"] == "boto3"

    releases = json_module.loads((out_dir / "release.json").read_text(encoding="utf-8"))
    assert releases[0]["summary"] == "x" * 509 + "..."
    assert len(releases[0]["summary"]) == 512

    maintainers = json_module.loads((out_dir / "maintainer.json").read_text(encoding="utf-8"))
    usernames = {m["username"]: m["joined_at"] for m in maintainers}
    assert usernames["alice"] == "2020-01-02T00:00:00+0000"
    assert usernames["bob"] is None  # placeholder hasil backfill

    captured = capsys.readouterr()
    assert "[1/15] Membersihkan package.json..." in captured.out
    assert "Lewati baris package.json" in captured.out
    assert "Selesai." in captured.out


def test_run_returns_1_when_no_run_folder_found(tmp_path, capsys):
    data_root = tmp_path / "001_02_data"
    data_root.mkdir()
    output_root = tmp_path / "003_02_data_cleaned"

    exit_code = run(data_root, output_root)

    assert exit_code == 1
    captured = capsys.readouterr()
    assert "berhenti karena error" in captured.err
