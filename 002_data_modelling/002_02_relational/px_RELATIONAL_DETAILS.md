# Relational Schema untuk PyPI Extraction Data

Model relasional hasil konversi dari ERD di `data_modelling/erd/px_ERD_DETAILS.md`.

## 1. Relasi dari Strong Entity

Strong entity direduksi jadi relasi dengan atribut yang sama persis (atribut simple-nya saja, atribut derived dibuang).

### RELASI: package

| Atribut | Tipe Data | Nullable | Default | Unique | Check | Klasifikasi | Asal |
|---|---|---|---|---|---|---|---|
| <u>`package_id`</u> | `UUID` | NOT NULL | `uuidv7()` | - | - | PK (_surrogate_) | bukan hasil mapping langsung dari ERD, lihat catatan dibawah |
| `org_id` | `UUID` | NULL | - | - | - | FK → `organization(organization_id)` | hasil merge relationship `owned_by` (PACKAGE N:1 ORGANIZATION). Karena relationship-nya bukan identifying dan berkardinalitas N:1, sisi N (PACKAGE) cukup diberi kolom tambahan berisi primary key sisi 1 (ORGANIZATION), tidak perlu relasi terpisah. Nullable karena partisipasi PACKAGE terhadap `owned_by` parsial (ada package tanpa organization) |
| `lifecycle_status` | `ENUM('archived','deprecated','quarantined')` | NULL | - | - | - | simple | atribut simple `lifecycle_status` milik entity PACKAGE |
| `name` | `TEXT` | NOT NULL | - | Ya | - | key, diturunkan jadi candidate key | atribut key `name` milik entity PACKAGE |

**Catatan:**
1. Karena PACKAGE direferensikan berulang oleh `release` dan `maintained_by`, sedangkan `name` bertipe `TEXT` (variable length, berpotensi panjang), foreign key yang menyalin `name` ke banyak baris anak akan lebih boros storage & index dibanding satu kolom UUID, sehingga keputusan akhir PACKAGE diberi surrogate key `package_id` sebagai primary key, sedangkan `name` tetap dipertahankan sebagai candidate key.

### RELASI: organization

| Atribut | Tipe Data | Nullable | Default | Unique | Check | Klasifikasi | Asal |
|---|---|---|---|---|---|---|---|
| <u>`organization_id`</u> | `UUID` | NOT NULL | `uuidv7()` | - | - | PK (_surrogate_) | bukan hasil mapping langsung dari ERD, lihat catatan dibawah |
| `display_name` | `VARCHAR(255)` | NOT NULL | - | - | - | simple | atribut simple `display_name` milik entity ORGANIZATION |
| `name` | `TEXT` | NOT NULL | - | Ya | - | key, diturunkan jadi candidate key | atribut key `name` milik entity ORGANIZATION |

**Catatan:**
1. Karena ORGANIZATION direferensikan oleh `package` dan satu organization bisa memiliki banyak package sekaligus, sehingga keputusan akhir ORGANIZATION diberi surrogate key `organization_id` sebagai primary key dengan alasan yang sama seperti PACKAGE, sedangkan `name` tetap dipertahankan sebagai candidate key.

### RELASI: maintainer

| Atribut | Tipe Data | Nullable | Default | Unique | Check | Klasifikasi | Asal |
|---|---|---|---|---|---|---|---|
| <u>`maintainer_id`</u> | `UUID` | NOT NULL | `uuidv7()` | - | - | PK (_surrogate_) | bukan hasil mapping langsung dari ERD, lihat catatan dibawah |
| `joined_at` | `TIMESTAMPTZ` | NULL | - | - | - | simple | atribut simple `joined_at` milik entity MAINTAINER |
| `username` | `TEXT COLLATE case_insensitive` | NOT NULL | - | Ya | `length(username) <= 50` | key, diturunkan jadi candidate key | atribut key `username` milik entity MAINTAINER |

**Catatan:**
1. Karena MAINTAINER direferensikan berulang oleh `maintained_by` (satu maintainer bisa terhubung ke banyak package), sehingga keputusan akhir MAINTAINER diberi surrogate key `maintainer_id` sebagai primary key dengan alasan yang sama seperti PACKAGE, sedangkan `username` tetap dipertahankan sebagai candidate key.
2. `joined_at` diubah dari NOT NULL menjadi NULLABLE berdasarkan verifikasi langsung ke live `pypi.org/user/<username>/`: sebagian akun (contoh akun milik organisasi seperti `aws`, atau sebagian akun individual lama seperti `dstufft`) tidak menampilkan metadiv "Date joined" sama sekali di halaman profilnya, bukan cuma tersembunyi lewat parsing yang salah. Karena `pxs` (scraper) tidak boleh memfabrikasi nilai yang sumbernya sendiri tidak menyediakannya, kolom ini WAJIB nullable.

### RELASI: classifier

| Atribut | Tipe Data | Nullable | Default | Unique | Check | Klasifikasi | Asal |
|---|---|---|---|---|---|---|---|
| <u>`classifier_id`</u> | `INTEGER` | NOT NULL | `GENERATED ALWAYS AS IDENTITY` | - | - | PK (_surrogate_) | bukan hasil mapping langsung dari ERD, lihat catatan dibawah |
| `category` | `VARCHAR(50)` | NOT NULL | - | Ya, bersama `value` | - | key (bagian dari composite key), diturunkan jadi candidate key | atribut key `category` milik entity CLASSIFIER |
| `value` | `VARCHAR(255)` | NOT NULL | - | Ya, bersama `category` | - | key (bagian dari composite key), diturunkan jadi candidate key | atribut key `value` milik entity CLASSIFIER |

**Catatan:**
1. Karena CLASSIFIER dipakai berulang sebagai sisi relationship M:N `tagged_with` dengan volume baris besar (banyak release, tiap release banyak classifier, satu classifier dipakai ulang oleh banyak release), FK komposit (`category`, `value`) berarti dua kolom string ikut disalin di setiap baris `tagged_with`, lebih boros storage & index dibanding satu kolom integer, sehingga keputusan akhir CLASSIFIER diberi surrogate key `classifier_id` sebagai primary key, sedangkan (`category`, `value`) tetap dipertahankan sebagai candidate key, bukan composite primary key seperti hasil mapping ERD langsung.
2. Karena CLASSIFIER adalah tabel referensi kecil dan tetap (finite, jumlah barisnya terbatas dan jarang bertambah), sehingga keputusan akhir tipe surrogate-nya `INTEGER`, bukan `UUID` seperti relasi lain, karena overhead 16-byte per baris pada `UUID` tidak sebanding dengan manfaatnya untuk tabel referensi sekecil ini.

### RELASI: attestation

| Atribut | Tipe Data | Nullable | Default | Unique | Check | Klasifikasi | Asal |
|---|---|---|---|---|---|---|---|
| <u>`sigstore_log_index`</u> | `BIGINT` | NOT NULL | - | - | - | PK | atribut key `sigstore_log_index` milik entity ATTESTATION |
| `release_file_id` | `UUID` | NOT NULL | - | - | - | FK → `release_file(release_file_id)` | hasil merge relationship `proves` (RELEASE_FILE 1:N ATTESTATION). Karena kardinalitasnya 1:N dan partisipasi ATTESTATION (sisi N) terhadap `proves` total, foreign key ke RELEASE_FILE digabungkan langsung ke relasi ATTESTATION (sisi banyak) |
| `integration_time` | `TIMESTAMPTZ` | NOT NULL | - | - | - | simple | atribut simple milik entity ATTESTATION |
| `statement_type` | `VARCHAR(255)` | NOT NULL | - | - | - | simple | atribut simple milik entity ATTESTATION |
| `predicate_type` | `VARCHAR(255)` | NOT NULL | - | - | - | simple | atribut simple milik entity ATTESTATION |
| `subject_name` | `VARCHAR(255)` | NOT NULL | - | - | - | simple | atribut simple milik entity ATTESTATION |
| `subject_digest` | `VARCHAR(64)` | NOT NULL | - | - | - | simple | atribut simple milik entity ATTESTATION |
| `source_repo` | `TEXT COLLATE case_insensitive` | NULL | - | - | - | simple | atribut simple milik entity ATTESTATION |
| `source_reference` | `VARCHAR(255)` | NULL | - | - | - | simple | atribut simple milik entity ATTESTATION |
| `token_issuer` | `VARCHAR(255)` | NOT NULL | - | - | - | simple | atribut simple milik entity ATTESTATION |
| `runner_environment` | `VARCHAR(50)` | NULL | - | - | - | simple | atribut simple milik entity ATTESTATION |
| `publisher_workflow` | `VARCHAR(255)` | NULL | - | - | - | simple | atribut simple milik entity ATTESTATION |
| `trigger_event` | `VARCHAR(50)` | NULL | - | - | - | simple | atribut simple milik entity ATTESTATION |

## 2. Relasi dari Weak Entity

Weak entity direduksi jadi relasi yang memuat primary key owner-nya digabung dengan discriminator dan atribut simple milik weak entity itu sendiri.

### RELASI: release

> Weak entity RELEASE, owner PACKAGE

| Atribut | Tipe Data | Nullable | Default | Unique | Check | Klasifikasi | Asal |
|---|---|---|---|---|---|---|---|
| <u>`release_id`</u> | `UUID` | NOT NULL | `uuidv7()` | - | - | PK (_surrogate_) | bukan hasil mapping langsung dari ERD, lihat catatan dibawah |
| `package_id` | `UUID` | NOT NULL | - | Ya, bersama `version` | - | FK, diturunkan jadi candidate key | primary key owner PACKAGE, masuk ke sini karena relationship identifying `has` (PACKAGE 1:N RELEASE) |
| `created` | `TIMESTAMPTZ` | NOT NULL | - | - | - | simple | atribut simple milik entity RELEASE |
| `is_prerelease` | `BOOLEAN` | NOT NULL | `false` | - | - | simple | atribut simple milik entity RELEASE |
| `yanked` | `BOOLEAN` | NOT NULL | `false` | - | - | simple | atribut simple milik entity RELEASE |
| `lifecycle_status` | `ENUM('archived','deprecated','quarantined')` | NULL | - | - | - | simple | atribut simple milik entity RELEASE |
| `version` | `TEXT` | NOT NULL | - | Ya, bersama `package_id` | - | discriminator, diturunkan jadi candidate key | discriminator entity RELEASE |
| `yanked_reason` | `TEXT` | NULL | - | - | - | simple | atribut simple milik entity RELEASE |
| `summary` | `VARCHAR(512)` | NULL | - | - | - | simple | atribut simple milik entity RELEASE |
| `license` | `TEXT` | NULL | - | - | - | simple | atribut simple milik entity RELEASE |
| `requires_python` | `TEXT` | NULL | - | - | - | simple | atribut simple milik entity RELEASE |

**Catatan:**
1. Karena relasi `release` diacu oleh enam relasi anak sekaligus (`release_detail`, `release_file`, `project_link`, `release_keyword`, `release_extra`, `tagged_with`), primary key komposit (`package_id`, `version`) berarti dua kolom ikut disalin berulang di keenam relasi tersebut, sehingga keputusan akhir RELEASE diberi surrogate key `release_id` sebagai primary key, sedangkan (`package_id`, `version`) tetap dipertahankan sebagai candidate key, bukan composite primary key seperti hasil mapping ERD langsung.
2. Karena `description` menyimpan HTML hasil render project description yang ukurannya besar dan sangat bervariasi antar release, sementara mayoritas akses ke `release` DIASUMSIKAN jarang atau bahkan tidak pernah pernah butuh kolom ini, menggabungkannya dapat membuat row bloat dan memperlambat scan pada kolom yang sering dipakai, sehingga keputusan akhir `description` dan beberapa data lain, seperti `meta_author`, `meta_author_email`, `meta_author_email_verified`, `meta_maintainer`, `meta_maintainer_email`, `meta_maintainer_email_verified` dipisah (_vertical partitioning_) ke relasi baru `release_detail` yang berelasi 1:1 terhadap `release` lewat primary key yang sama, bukan digabung jadi satu relasi besar seperti hasil mapping ERD langsung.

### RELASI: release_detail

> hasil vertical partitioning dari `release`, bukan mapping langsung ERD

| Atribut | Tipe Data | Nullable | Default | Unique | Check | Klasifikasi | Asal |
|---|---|---|---|---|---|---|---|
| <u>`release_id`</u> | `UUID` | NOT NULL | - | - | - | PK, FK → `release(release_id)` | primary key `release` |
| `meta_author_email_verified` | `BOOLEAN` | NULL | `false` | - | - | simple | atribut simple milik entity RELEASE, dipindah ke sini untuk optimasi basis data |
| `meta_maintainer_email_verified` | `BOOLEAN` | NULL | `false` | - | - | simple | atribut simple milik entity RELEASE, dipindah ke sini untuk optimasi basis data |
| `description` | `TEXT` | NULL | - | - | - | simple | atribut simple `description` milik entity RELEASE, dipindah ke sini untuk optimasi basis data |
| `meta_author` | `TEXT` | NULL | - | - | - | simple | atribut simple milik entity RELEASE, dipindah ke sini untuk optimasi basis data |
| `meta_author_email` | `TEXT COLLATE case_insensitive` | NULL | - | - | - | simple | atribut simple milik entity RELEASE, dipindah ke sini untuk optimasi basis data |
| `meta_maintainer` | `TEXT` | NULL | - | - | - | simple | atribut simple milik entity RELEASE, dipindah ke sini untuk optimasi basis data |
| `meta_maintainer_email` | `TEXT COLLATE case_insensitive` | NULL | - | - | - | simple | atribut simple milik entity RELEASE, dipindah ke sini untuk optimasi basis data |

### RELASI: release_file

> Weak entity RELEASE_FILE, owner RELEASE

| Atribut | Tipe Data | Nullable | Default | Unique | Check | Klasifikasi | Asal |
|---|---|---|---|---|---|---|---|
| <u>`release_file_id`</u> | `UUID` | NOT NULL | `uuidv7()` | - | - | PK (_surrogate_) | bukan hasil mapping langsung dari ERD, lihat catatan dibawah |
| `release_id` | `UUID` | NOT NULL | - | Ya, bersama `filename` | - | FK, diturunkan jadi candidate key | primary key owner RELEASE, masuk karena relationship identifying `distributes` (RELEASE 1:N RELEASE_FILE) |
| `size` | `BIGINT` | NOT NULL | - | - | - | simple | atribut simple milik entity RELEASE_FILE |
| `upload_time` | `TIMESTAMPTZ` | NOT NULL | - | - | - | simple | atribut simple milik entity RELEASE_FILE |
| `is_trusted_publishing` | `BOOLEAN` | NOT NULL | `false` | - | - | simple | atribut simple milik entity RELEASE_FILE |
| `filename` | `TEXT` | NOT NULL | - | Ya, bersama `release_id` | - | discriminator, diturunkan jadi candidate key | discriminator entity RELEASE_FILE |
| `path` | `TEXT` | NOT NULL | - | - | - | simple | atribut simple milik entity RELEASE_FILE |
| `uploaded_via` | `VARCHAR(255)` | NULL | - | - | - | simple | atribut simple milik entity RELEASE_FILE |

**Catatan:**
1. Karena RELEASE_FILE jadi FK di tiga relasi anak (`file_hash`, `release_file_tag`, `attestation`) dan salah satu bagian primary key aslinya (`filename`) adalah string yang mungkin panjang, primary key komposit (`package_id`, `version`, `filename`) akan disalin tiga kolom sekaligus ke ketiga relasi tersebut, sehingga keputusan akhir RELEASE_FILE diberi surrogate key `release_file_id` sebagai primary key, sedangkan (`release_id`, `filename`) tetap dipertahankan sebagai candidate key, bukan composite primary key seperti hasil mapping ERD langsung.
2. Karena `{wheel_tag}` adalah atribut multivalued (satu file bisa punya banyak tag platform sekaligus), sehingga atribut tersebut dipecah ke relasi tersendiri (`release_file_tag`), bukan digabung ke `release_file`.

### RELASI: file_hash

> Weak entity FILE_HASH, owner RELEASE_FILE

| Atribut | Tipe Data | Nullable | Default | Unique | Check | Klasifikasi | Asal |
|---|---|---|---|---|---|---|---|
| <u>`release_file_id`</u> | `UUID` | NOT NULL | - | - | - | PK (turunan owner), FK → `release_file(release_file_id)` | primary key owner RELEASE_FILE, masuk karena relationship identifying `checks` (RELEASE_FILE 1:N FILE_HASH) |
| <u>`algorithm`</u> | `ENUM('SHA256','MD5','BLAKE2b-256')` | NOT NULL | - | - | - | PK (_discriminator_) | discriminator entity FILE_HASH |
| `digest` | `VARCHAR(64)` | NOT NULL | - | - | - | simple | atribut simple milik entity FILE_HASH |

**Catatan:**
1. Karena notasi relasional standar (PK/FK saja) tidak bisa menyatakan business constraint "tepat 3 baris per file" pada relationship `checks` (SHA256/MD5/BLAKE2b-256), sehingga tetap perlu constraint tambahan di luar DDL dasar (misalnya lewat trigger atau validasi di application layer).

### RELASI: project_link

> Weak entity PROJECT_LINK, owner RELEASE

| Atribut | Tipe Data | Nullable | Default | Unique | Check | Klasifikasi | Asal |
|---|---|---|---|---|---|---|---|
| <u>`release_id`</u> | `UUID` | NOT NULL | - | - | - | PK (turunan owner), FK → `release(release_id)` | primary key owner RELEASE, masuk karena relationship identifying `links` (RELEASE 1:N PROJECT_LINK) |
| <u>`label`</u> | `TEXT COLLATE case_insensitive` | NOT NULL | - | - | - | PK (_discriminator_) | discriminator entity PROJECT_LINK |
| `verified` | `BOOLEAN` | NOT NULL | `false` | - | - | simple | atribut simple milik entity PROJECT_LINK |
| `url` | `TEXT` | NOT NULL | - | - | - | simple | atribut simple milik entity PROJECT_LINK |

## 3. Relasi dari Relationship M:N

Relationship many-to-many tidak bisa digabung ke salah satu sisi (beda dengan relationship 1:N atau N:1 di atas), jadi tetap direduksi jadi relasi tersendiri berisi primary key kedua entity yang terlibat plus atribut deskriptif relationship-nya (kalau ada).

### RELASI: maintained_by

| Atribut | Tipe Data | Nullable | Default | Unique | Check | Klasifikasi | Asal |
|---|---|---|---|---|---|---|---|
| <u>`package_id`</u> | `UUID` | NOT NULL | - | - | - | PK, FK → `package(package_id)` | relationship M:N `maintained_by` sisi PACKAGE |
| <u>`maintainer_id`</u> | `UUID` | NOT NULL | - | - | - | PK, FK → `maintainer(maintainer_id)` | relationship M:N `maintained_by` sisi MAINTAINER |

### RELASI: tagged_with

| Atribut | Tipe Data | Nullable | Default | Unique | Check | Klasifikasi | Asal |
|---|---|---|---|---|---|---|---|
| <u>`release_id`</u> | `UUID` | NOT NULL | - | - | - | PK, FK → `release(release_id)` | relationship M:N `tagged_with` sisi RELEASE |
| <u>`classifier_id`</u> | `INTEGER` | NOT NULL | - | - | - | PK, FK -> `classifier(classifier_id)` | relationship M:N `tagged_with` sisi CLASSIFIER |

## 4. Relasi dari Multivalued Attribute

Tiap atribut multivalued direduksi jadi relasi baru berisi primary key entity pemiliknya digabung dengan atribut multivalued itu sendiri, karena satu entity bisa punya banyak nilai untuk atribut yang sama.

### RELASI: release_keyword

| Atribut | Tipe Data | Nullable | Default | Unique | Check | Klasifikasi | Asal |
|---|---|---|---|---|---|---|---|
| <u>`release_id`</u> | `UUID` | NOT NULL | - | - | - | PK, FK → `release(release_id)` | primary key entity RELEASE (pemilik atribut) |
| <u>`keyword`</u> | `VARCHAR(100)` | NOT NULL | - | - | - | PK | atribut multivalued `{keyword}` milik entity RELEASE |

### RELASI: release_extra

| Atribut | Tipe Data | Nullable | Default | Unique | Check | Klasifikasi | Asal |
|---|---|---|---|---|---|---|---|
| <u>`release_id`</u> | `UUID` | NOT NULL | - | - | - | PK, FK → `release(release_id)` | primary key entity RELEASE (pemilik atribut) |
| <u>`extra_name`</u> | `VARCHAR(100)` | NOT NULL | - | - | - | PK | atribut multivalued `{extra_name}` milik entity RELEASE |

### RELASI: release_file_tag

| Atribut | Tipe Data | Nullable | Default | Unique | Check | Klasifikasi | Asal |
|---|---|---|---|---|---|---|---|
| <u>`release_file_id`</u> | `UUID` | NOT NULL | - | - | - | PK, FK → `release_file(release_file_id)` | primary key entity RELEASE_FILE (pemilik atribut) |
| <u>`wheel_tag`</u> | `VARCHAR(100)` | NOT NULL | - | - | - | PK | atribut multivalued `{wheel_tag}` milik entity RELEASE_FILE |

## 5. Relasi dari Derived Attribute

Atribut derived (`normalized_name()`, `owned_project_count()`, `maintained_project_count()`, `is_valid_spdx_license()`, `is_valid_requires_python()`, `packagetype()`) tidak dimasukkan ke skema manapun karena derived attribute tidak diimplementasikan sebagai kolom, CUKUP dihitung ulang saat dibutuhkan.

## 6. Mekanisme Umum yang Dipakai di Seluruh Relasi

1. **Case-insensitive tanpa extension `citext`.** Kolom yang butuh perbandingan case-insensitive (`username`, `meta_author_email`, `meta_maintainer_email`, `source_repo`, `label`) memakai `TEXT` biasa dengan custom collation ICU, bukan tipe `CITEXT`. Pada equality/range comparison (yang jadi kebutuhan utama kolom-kolom ini, bukan pattern matching `LIKE`), custom collation konsisten lebih cepat dibanding `CITEXT` terutama pada sequential scan. Setup satu kali di awal database:

   ```sql
   CREATE COLLATION case_insensitive (
       provider = icu, locale = 'und-u-ks-level2', deterministic = false
   );
   ```
   Lalu tinggal dipakai lewat `TEXT COLLATE case_insensitive` di kolom yang relevan.

2. **UUID versi 7, bukan versi 4.** Semua surrogate key `UUID` memakai default `uuidv7()`, bukan `gen_random_uuid()` (UUID v4/acak). UUIDv7 memuat komponen timestamp sehingga nilai baru cenderung berurutan, membuat insert baru selalu jatuh di ujung kanan B-tree index (mirip `BIGINT` auto-increment), bukan menyebar acak. Untuk tabel yang terus tumbuh, ini mengurangi index bloat dan mempercepat range scan.