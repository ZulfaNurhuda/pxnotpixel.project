"""Mesin cleaning generik berbasis schema.SCHEMA — satu implementasi untuk semua entity."""
from schema import SCHEMA


class CleaningReport:
    """Akumulasi & laporkan langsung apa yang terjadi selama proses cleaning."""

    def __init__(self):
        self.dropped_count = 0
        self.truncated_count = 0
        self.backfilled_count = 0

    def drop(self, entity_filename, reason, row):
        self.dropped_count += 1
        print(f"Lewati baris {entity_filename} ({reason}): {row}")

    def truncate(self, count=1):
        self.truncated_count += count

    def backfill(self):
        self.backfilled_count += 1


def _is_missing(value):
    return value is None or value == ""


def _clean_row(entity_filename, row, columns, report):
    cleaned_row = {}
    for name, spec in columns.items():
        value = row.get(name)

        if _is_missing(value):
            if not spec["nullable"]:
                report.drop(entity_filename, f"{name} kosong tapi wajib diisi", row)
                return None
            cleaned_row[name] = None
            continue

        if spec["enum"] is not None and value not in spec["enum"]:
            report.drop(entity_filename, f"{name} tidak valid: {value!r}", row)
            return None

        if spec["max_length"] is not None and isinstance(value, str) and len(value) > spec["max_length"]:
            if spec["kind"] == "identifier":
                report.drop(entity_filename, f"{name} kepanjangan", row)
                return None
            value = value[: spec["max_length"] - 3] + "..."
            report.truncate()

        cleaned_row[name] = value

    return cleaned_row


def clean_entity(entity_filename, rows, report):
    """Baris yang gagal cek NOT NULL/enum/panjang-identifier di-drop; kolom deskriptif kepanjangan dipotong, bukan di-drop."""
    columns = SCHEMA[entity_filename]
    cleaned = []
    for row in rows:
        cleaned_row = _clean_row(entity_filename, row, columns, report)
        if cleaned_row is not None:
            cleaned.append(cleaned_row)
    return cleaned


def backfill_maintainers(maintainer_rows, maintained_by_rows, report):
    """Tambahkan placeholder maintainer dengan joined_at=None (bukan dikarang) untuk maintainer yang profilnya gagal ke-fetch."""
    known = {m["username"].casefold() for m in maintainer_rows}
    result = list(maintainer_rows)
    for mb in maintained_by_rows:
        username = mb["maintainer_username"]
        key = username.casefold()
        if key in known:
            continue
        result.append({"username": username, "joined_at": None})
        known.add(key)
        report.backfill()
    return result
