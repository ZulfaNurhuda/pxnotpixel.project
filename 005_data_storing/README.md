# `005_data_storing`

Modul ini berisi definisi database Postgres tempat data PyPI yang sudah dibersihkan disimpan. Database dijalankan lewat Docker Compose, dengan skema dan seluruh tuning performa sudah didefinisikan di dalam folder [`005_01_docker/`](005_01_docker/compose.yml).

## Cara Menyalakan Database

Jalankan dari dalam folder `005_data_storing/`:

```bash
cp 005_01_docker/.env.example 005_01_docker/.env
```

`.env.example` ada di [`005_01_docker/.env.example`](005_01_docker/.env.example). Buka `005_01_docker/.env` (hasil copy di atas), isi nilai `POSTGRES_PASSWORD` dengan password sendiri, lalu jalankan:

```bash
docker compose -f 005_01_docker/compose.yml up -d
```

Saat container pertama kali dibuat, script di [`005_01_docker/init/01_px_INIT.sql`](005_01_docker/init/01_px_INIT.sql) otomatis dijalankan oleh image Postgres untuk membuat database, collation, fungsi, tipe enum, dan seluruh tabel. Script ini idempoten, jadi aman dijalankan ulang tanpa merusak data yang sudah ada.

## Detail Koneksi

| Parameter | Nilai |
|---|---|
| Host | `127.0.0.1` |
| Port | `5432` |
| User | `px_user` |
| Database | `px_db` |
| Password | sesuai isi `POSTGRES_PASSWORD` di `005_01_docker/.env` |

Port di-bind hanya ke `127.0.0.1`, jadi database tidak bisa diakses dari luar mesin tempat container berjalan.

## Ringkasan Skema

Skema terdiri dari 15 tabel. Detail lengkap tiap tabel, kolom, dan relasinya ada di [`002_data_modelling/README.md`](../002_data_modelling/README.md), tapi berikut ringkasan bagian-bagian penting yang perlu diketahui sebelum mengoperasikan database ini:

- Dua tipe ENUM kustom: `LIFECYCLE_STATUS_ENUM` (`archived`, `deprecated`, `quarantined`) dan `HASH_ALGORITHM_ENUM` (`SHA256`, `MD5`, `BLAKE2b-256`).
- Collation `case_insensitive`, dibuat dengan provider ICU, dipakai di kolom seperti username maintainer dan label project link yang perlu dicocokkan tanpa peduli besar-kecil huruf.
- Fungsi `uuidv7()` kustom untuk menghasilkan primary key UUIDv7 pada sebagian besar tabel. Ada dua pengecualian: tabel `classifier` memakai `INTEGER GENERATED ALWAYS AS IDENTITY`, dan tabel `attestation` memakai `sigstore_log_index` sebagai primary key natural (bukan UUID yang di-generate).

## Volume

Compose mendefinisikan bind mount berikut:

- `pg_data/`: lokasi data files Postgres di host, di-gitignore, dan akan tetap persisten selama folder ini tidak dihapus manual.
- `../005_02_export` ke `/export` di dalam container: supaya hasil `pg_dump` langsung muncul di `005_02_export/` pada host, tanpa perlu `docker cp` manual. Jalankan dari mana saja setelah container aktif:

  ```bash
  docker exec px_postgres pg_dump -U px_user -d px_db -f /export/px_DB_DUMP.sql
  ```

  File `px_DB_DUMP.sql` langsung tersedia di `005_data_storing/005_02_export/` begitu perintah di atas selesai.

## Tuning Performa

[`compose.yml`](005_01_docker/compose.yml) menyertakan beberapa parameter Postgres yang di-tuning untuk lingkungan development lokal, seperti `shared_buffers`, `work_mem`, `effective_cache_size`, dan beberapa parameter WAL/checkpoint lainnya. Nilai-nilai ini disesuaikan dengan batas `mem_limit` dan `cpus` container.
