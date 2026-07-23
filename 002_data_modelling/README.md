# `002_data_modelling`

Subsistem kedua pipeline `pxnotpixel`: pemodelan data dari hasil scraping `001_data_scraping` menjadi skema basis data relasional yang siap dipakai `004_data_loader`. Terdiri dari tiga tahap berurutan: ERD, skema relasional, dan pembuktian normalisasi.

## Pendekatan modelling

Struktur entity mengikuti struktur data yang benar-benar tersedia di halaman `pypi.org/project/<nama>/` dan `pypi.org/user/<username>/`. Entity utama yang dimodelkan: `PACKAGE` (satu nama project), `RELEASE` (satu versi rilis, tempat mayoritas metadata berada), `RELEASE_FILE` (file distribusi yang bisa di-download per rilis), `FILE_HASH` (checksum per file), `PROJECT_LINK` (link eksternal per rilis), `MAINTAINER` (akun PyPI yang mengelola package), `ORGANIZATION` (organisasi pemilik banyak package sekaligus), `CLASSIFIER` (tag Trove standar yang dishare lintas package), dan `ATTESTATION` (bukti provenance kriptografis per file).

Pemisahan `RELEASE` dari `PACKAGE`, dan `ORGANIZATION` dari `PACKAGE`, dilakukan karena satu package punya banyak versi rilis dengan metadata berbeda-beda, dan satu organisasi bisa memiliki banyak package sekaligus, menggabungkannya jadi atribut biasa akan mengulang nilai yang sama di banyak baris.

## 1. Entity Relationship Diagram

Ringkasan: [`002_01_erd/px_ERD_DETAILS.md`](002_01_erd/px_ERD_DETAILS.md), diagram lengkap: [`002_01_erd/px_ERD.png`](002_01_erd/px_ERD.png) / [`002_01_erd/px_ERD.svg`](002_01_erd/px_ERD.svg).

Setiap entity dan atribut di dokumen ERD diverifikasi lewat tiga sumber independen (template rendering resmi Warehouse, skema `warehouse/packaging/models.py`, dan HTML mentah live dari lima package berbeda karakter), sehingga hanya memuat data yang terbukti dirender di halaman web dan bisa diperoleh lewat scraping.

Poin utama:

- 9 entity: `PACKAGE`, `ORGANIZATION`, `RELEASE` (weak, owner `PACKAGE`), `RELEASE_FILE` (weak, owner `RELEASE`), `FILE_HASH` (weak, owner `RELEASE_FILE`), `PROJECT_LINK` (weak, owner `RELEASE`), `MAINTAINER`, `CLASSIFIER`, `ATTESTATION`. `ATTESTATION` tetap strong entity meski existence-dependent terhadap `RELEASE_FILE`, karena dia punya key natural sendiri (`sigstore_log_index`, diterbitkan Rekor, unik global di transparency log Sigstore).
- 8 relationship: `has` (`PACKAGE`-`RELEASE`, 1:N, identifying), `owned_by` (`PACKAGE`-`ORGANIZATION`, N:1), `distributes` (`RELEASE`-`RELEASE_FILE`, 1:N, identifying), `checks` (`RELEASE_FILE`-`FILE_HASH`, 1:N tepat 3, identifying), `links` (`RELEASE`-`PROJECT_LINK`, 1:N, identifying), `proves` (`RELEASE_FILE`-`ATTESTATION`, 1:N), `maintained_by` (`PACKAGE`-`MAINTAINER`, M:N), `tagged_with` (`RELEASE`-`CLASSIFIER`, M:N).
- Atribut kunci: `PACKAGE.name`, `ORGANIZATION.name`, `MAINTAINER.username` sebagai key sederhana. `RELEASE.version` dan `RELEASE_FILE.filename` sebagai discriminator (weak entity). `CLASSIFIER` pakai composite key (`category`, `value`). `ATTESTATION.sigstore_log_index` sebagai key.
- Beberapa pasangan atribut yang sekilas mirip composite attribute (`meta_author`/`meta_author_email`, `yanked`/`yanked_reason`) dimodelkan sebagai simple attribute terpisah, karena bukan pecahan dari satu nilai utuh yang sama, melainkan fakta independen atau pola flag-dan-detail-kondisional.

## 2. Skema Relasional

Ringkasan: [`002_02_relational/px_RELATIONAL_DETAILS.md`](002_02_relational/px_RELATIONAL_DETAILS.md), diagram lengkap: [`002_02_relational/px_RELATIONAL.png`](002_02_relational/px_RELATIONAL.png) / [`002_02_relational/px_RELATIONAL.svg`](002_02_relational/px_RELATIONAL.svg).

Poin utama:

- Strong entity direduksi langsung jadi tabel dengan atribut simple-nya (`package`, `organization`, `maintainer`, `classifier`, `attestation`), atribut derived (`normalized_name()`, `owned_project_count()`, dan sejenisnya) tidak diimplementasikan sebagai kolom, cukup dihitung ulang saat dibutuhkan.
- Weak entity direduksi jadi tabel yang memuat primary key owner digabung discriminator dan atribut simple sendiri (`release`, `release_file`, `file_hash`, `project_link`).
- `package`, `organization`, `maintainer`, `release`, `release_file` memakai surrogate key `UUID` dengan default `uuidv7()` (bukan versi 4 acak), sedangkan atribut key ERD aslinya (`name`, `username`, pasangan `package_id`+`version`, dst) tetap dipertahankan sebagai candidate key lewat constraint `UNIQUE`. Alasannya: kolom-kolom itu direferensikan berulang oleh tabel anak, dan tipe aslinya (`TEXT`) lebih boros storage/index dibanding satu kolom `UUID` saat disalin ke banyak baris FK. `classifier` memakai surrogate `INTEGER` (bukan `UUID`) karena tabel referensi kecil dan tetap.
- `release` dipartisi vertikal menjadi `release` dan `release_detail` (relasi 1:1 lewat primary key yang sama), memisahkan kolom besar dan jarang diakses (`description`, `meta_author*`, `meta_maintainer*`) dari kolom yang sering di-scan.
- Relationship M:N (`maintained_by`, `tagged_with`) tetap jadi tabel junction tersendiri berisi pasangan foreign key kedua sisi.
- Atribut multivalued (`{keyword}`, `{extra_name}`, `{wheel_tag}`) masing-masing jadi tabel tersendiri (`release_keyword`, `release_extra`, `release_file_tag`) berisi FK ke pemiliknya digabung nilai atribut itu sendiri.
- Constraint penting yang berulang di banyak tabel: kolom yang butuh perbandingan case-insensitive (`username`, `meta_author_email`, `meta_maintainer_email`, `source_repo`, `label`) memakai `TEXT COLLATE case_insensitive`, custom collation ICU (`provider = icu, locale = 'und-u-ks-level2', deterministic = false`) yang dibuat sekali di awal database, bukan tipe `CITEXT`, karena lebih cepat untuk equality/range comparison terutama pada sequential scan.
- Business constraint yang tidak bisa dinyatakan lewat notasi relasional standar (misalnya `checks` harus tepat 3 baris per file: SHA256, MD5, BLAKE2b-256) dicatat sebagai constraint tambahan di luar DDL dasar (trigger atau validasi application layer).

## 3. Pembuktian Normalisasi

Dokumen lengkap: [`002_03_norm/px_NORMALIZATION_DETAILS.md`](002_03_norm/px_NORMALIZATION_DETAILS.md).

Seluruh 15 tabel hasil skema relasional (`organization`, `maintainer`, `classifier`, `package`, `release`, `release_detail`, `release_file`, `file_hash`, `project_link`, `maintained_by`, `tagged_with`, `release_keyword`, `release_extra`, `release_file_tag`, `attestation`) terbukti berada di BCNF, dicek lewat functional dependency (FD) dan candidate key masing-masing.

Contoh pembuktian singkat (tabel `release_file`): FD-nya adalah `release_file_id -> release_id, size, upload_time, is_trusted_publishing, filename, path, uploaded_via` dan `release_id, filename -> release_file_id, size, upload_time, is_trusted_publishing, path, uploaded_via`. Kedua super key (`release_file_id` dan pasangan `release_id`+`filename`) muncul sebagai LHS pada seluruh FD non-trivial, tidak ada FD dengan LHS yang bukan superkey, sehingga tabel ini BCNF.

Tabel junction dan tabel hasil pemecahan atribut multivalued (`maintained_by`, `tagged_with`, `release_keyword`, `release_extra`, `release_file_tag`) tidak punya atribut non-key sama sekali, sehingga BCNF secara trivial (tidak mungkin ada pelanggaran bentuk normal kalau tidak ada atribut non-key yang bisa jadi target FD).

## Translasi ERD ke skema relasional

Alur konversi mengikuti aturan mapping standar ERD-ke-relasional:

1. **Entity ke tabel.** Strong entity langsung jadi tabel dengan atribut simple-nya. Weak entity jadi tabel yang memuat primary key tabel owner-nya digabung discriminator dan atribut simple sendiri, karena weak entity tidak punya identitas independen dari owner-nya.
2. **Relationship 1:N / N:1 ke foreign key.** Relationship non-identifying berkardinalitas 1:N (misalnya `owned_by` antara `PACKAGE` dan `ORGANIZATION`) cukup ditambahkan sebagai kolom foreign key di sisi N, tanpa perlu tabel terpisah. Relationship identifying (misalnya `has` antara `PACKAGE` dan `RELEASE`) sekaligus membentuk sebagian primary key tabel weak entity di sisi turunannya.
3. **Relationship M:N ke tabel junction.** Relationship M:N tidak bisa digabung ke salah satu sisi karena tidak ada sisi yang secara fungsional menentukan sisi lain, sehingga tetap direduksi jadi tabel tersendiri berisi pasangan foreign key kedua entity yang terlibat. Ini yang menghasilkan `maintained_by` (dari relationship M:N `PACKAGE`-`MAINTAINER`) dan `tagged_with` (dari relationship M:N `RELEASE`-`CLASSIFIER`).
4. **Atribut ke kolom dengan tipe data.** Atribut simple jadi kolom dengan tipe data yang dipilih berdasarkan sifat datanya (misalnya `TEXT` untuk string panjang variatif, `TIMESTAMPTZ` untuk waktu, `BOOLEAN` untuk flag, `ENUM` untuk nilai terbatas seperti `lifecycle_status`). Atribut multivalued dipecah jadi tabel tersendiri berisi foreign key ke pemiliknya digabung nilai atributnya. Atribut derived tidak dipetakan jadi kolom sama sekali, karena nilainya dihitung ulang saat dibutuhkan, bukan disimpan.
5. **Penyesuaian di luar mapping langsung.** Beberapa keputusan (surrogate key `UUID`/`INTEGER` menggantikan candidate key natural, vertical partitioning `release`/`release_detail`) bukan hasil mapping ERD langsung, melainkan optimasi tambahan berdasarkan pola penggunaan (frekuensi referensi dari tabel anak, ukuran dan frekuensi akses kolom). Detail alasan tiap keputusan ada di catatan masing-masing tabel dalam [`002_02_relational/px_RELATIONAL_DETAILS.md`](002_02_relational/px_RELATIONAL_DETAILS.md).
