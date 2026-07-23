"""Entrypoint 003_data_transformer: cari run scraping terbaru, baca/tulis file entity JSON."""
import json
import sys
from pathlib import Path

import pandas as pd

from clean import CleaningReport, backfill_maintainers, clean_entity
from schema import ENTITY_FILES


def find_latest_run(data_root: Path) -> Path:
    """Folder pxs_<timestamp> terbaru; nama ISO 8601 basic sort benar secara string."""
    candidates = sorted(p for p in data_root.glob("pxs_*") if p.is_dir())
    if not candidates:
        raise FileNotFoundError(f"tidak ada folder hasil scraping ditemukan di {data_root}")
    return candidates[-1]


def read_entity(run_dir: Path, entity_filename: str) -> list:
    # convert_dates=False: cegah pandas auto-parse kolom tanggal jadi Timestamp yang merusak string ISO 8601 asli.
    df = pd.read_json(run_dir / entity_filename, convert_dates=False)
    # cast ke object dulu agar None tidak ke-coerce jadi NaN oleh dtype "str" pandas 3.x saat .where().
    df = df.astype(object).where(pd.notnull(df), None)
    return df.to_dict(orient="records")


def write_entity(out_dir: Path, entity_filename: str, rows: list) -> None:
    with open(out_dir / entity_filename, "w", encoding="utf-8") as f:
        json.dump(rows, f, ensure_ascii=False, separators=(",", ":"))


def run(data_root: Path, output_root: Path) -> int:
    try:
        run_dir = find_latest_run(data_root)
    except FileNotFoundError as e:
        print(f"003_data_transformer berhenti karena error: {e}", file=sys.stderr)
        return 1

    out_dir = output_root / run_dir.name

    report = CleaningReport()
    cleaned = {}

    total = len(ENTITY_FILES)
    for i, entity_filename in enumerate(ENTITY_FILES, start=1):
        print(f"[{i}/{total}] Membersihkan {entity_filename}...")
        entity_path = run_dir / entity_filename
        if not entity_path.exists():
            print(
                f"003_data_transformer berhenti karena error: file {entity_filename} tidak ditemukan di {run_dir}",
                file=sys.stderr,
            )
            return 1
        rows = read_entity(run_dir, entity_filename)
        cleaned[entity_filename] = clean_entity(entity_filename, rows, report)

    cleaned["maintainer.json"] = backfill_maintainers(
        cleaned["maintainer.json"], cleaned["maintained_by.json"], report
    )

    out_dir.mkdir(parents=True, exist_ok=True)

    for entity_filename, rows in cleaned.items():
        write_entity(out_dir, entity_filename, rows)

    print(f"{report.dropped_count} baris dilewati karena error, {report.truncated_count} baris dipotong (truncate).")
    print(f"{report.backfilled_count} maintainer placeholder dibuat (profil gak sempat ke-fetch pas scraping).")
    print("Selesai. Data hasil cleaning sudah ditulis.")
    return 0


def main() -> int:
    script_dir = Path(__file__).resolve().parent
    project_root = script_dir.parent.parent
    data_root = project_root / "001_data_scraping" / "001_02_data"
    output_root = script_dir.parent / "003_02_data_cleaned"
    return run(data_root, output_root)


if __name__ == "__main__":
    sys.exit(main())
