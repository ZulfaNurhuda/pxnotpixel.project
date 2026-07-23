# Entity Relationship Diagram untuk PyPI Extraction Data

Model ERD untuk data yang diambil dari halaman web PyPI (`pypi.org/project/<nama>/`). Setiap entity dan atribut di dokumen ini terverifikasi dengan tiga sumber:

1. Template rendering resmi Warehouse (repo [`pypi/warehouse`](https://github.com/pypi/warehouse)):
   - [`warehouse/templates/packaging/detail.html`](https://github.com/pypi/warehouse/blob/main/warehouse/templates/packaging/detail.html)
   - [`warehouse/templates/includes/packaging/project-data.html`](https://github.com/pypi/warehouse/blob/main/warehouse/templates/includes/packaging/project-data.html)
   - [`warehouse/templates/includes/file-details.html`](https://github.com/pypi/warehouse/blob/main/warehouse/templates/includes/file-details.html)
   - [`warehouse/templates/accounts/profile.html`](https://github.com/pypi/warehouse/blob/main/warehouse/templates/accounts/profile.html)
2. Skema database: [`warehouse/packaging/models.py`](https://github.com/pypi/warehouse/blob/main/warehouse/packaging/models.py)
3. HTML mentah live page lima package berbeda karakter: [`packaging`](https://pypi.org/project/packaging/), [`numpy`](https://pypi.org/project/numpy/), [`requests`](https://pypi.org/project/requests/), [`Flask`](https://pypi.org/project/Flask/), [`pytest`](https://pypi.org/project/pytest/)

Dokumen ini hanya memuat data yang terbukti dirender di halaman web, sehingga seluruhnya bisa diperoleh lewat scraping.

## Entity Set

### PACKAGE (_strong entity_)

**Merepresentasikan apa?** Satu nama project di PyPI.

| Atribut | Klasifikasi | Bukti render |
|---|---|---|
| `name` | key, simple | header halaman, juga bagian URL `pypi.org/project/<nama>/` |
| `lifecycle_status` | simple, nullable | badge "quarantined" / callout "archived" |
| `normalized_name()` | derived | dihitung dari `name` dengan aturan normalisasi PEP 503, terlihat juga di URL |

### ORGANIZATION (_strong entity_)

**Merepresentasikan apa?** Satu organisasi yang bisa memiliki banyak package sekaligus (1:N ke PACKAGE). Dipisah dari PACKAGE karena satu organization bisa memiliki banyak project (misalnya org besar seperti NumPy dan PyPA), jadi kalau digabung jadi atribut biasa, nama organisasi bakal keulang-ulang di tiap row PACKAGE.

| Atribut | Klasifikasi | Bukti render |
|---|---|---|
| `name` | key, simple | bagian URL `/org/<name>/` |
| `display_name` | simple | teks yang tampil di heading "Owner" |
| `owned_project_count()` | derived | dihitung dari COUNT relationship `owned_by` |

### RELEASE (_weak entity_, owner: PACKAGE, discriminator: `version`)

**Merepresentasikan apa?** Satu versi rilis spesifik dari sebuah package. Mayoritas metadata halaman (author, license, summary, dst) berada di level ini, bukan di PACKAGE.

| Atribut | Klasifikasi | Bukti render |
|---|---|---|
| `version` | discriminator | header + release history |
| `created` | simple | "Released: ..." + tanggal per versi di release history |
| `is_prerelease` | simple | badge "pre-release" |
| `yanked` | simple | badge "yanked" |
| `yanked_reason` | simple, nullable | callout "Reason this release was yanked" |
| `lifecycle_status` | simple, nullable | badge "This release has been quarantined" (independen dari status project) |
| `summary` | simple, nullable | tagline di bawah header |
| `description` | simple, nullable | tab "Project description" (HTML hasil render; bentuk mentah, content type, dan engine render tidak pernah ditampilkan halaman) |
| `license` | simple, nullable | Meta "License Expression:" (SPDX) atau "License:" (teks bebas legacy), dua-duanya mutually exclusive |
| `is_valid_spdx_license()` | derived | diproses dari `license` lewat validasi SPDX |
| `meta_author` | simple, nullable | Meta "Author:" |
| `meta_author_email` | simple, nullable | atribut `mailto:` pada Author |
| `meta_author_email_verified` | simple, nullable | Section tempat muncul (Verified vs Unverified details) |
| `meta_maintainer` | simple, nullable | Meta "Maintainer:" (string self-declared dari metadata upload, berbeda dari entity MAINTAINER) |
| `meta_maintainer_email` | simple, nullable | atribut `mailto:` pada Maintainer |
| `meta_maintainer_email_verified` | simple, nullable | Section tempat muncul (Verified vs Unverified details) |
| `requires_python` | simple, nullable | Meta "Requires: Python ..." |
| `is_valid_requires_python()` | derived | diproses dari `requires_python` lewat validasi format PEP 440 |
| `{keyword}` | multivalued | baris tags di Meta |
| `{extra_name}` | multivalued | Meta "Provides-Extra:" |

Catatan:
1. Karena syarat composite attribute itu sub-atributnya harus jadi pecahan yang kalau **digabung membentuk satu nilai utuh sejenis** (kayak `first_name`+`last_name` = satu nama utuh), dan `name` dengan `email` bukan pecahan dari satu nilai yang sama (gak ada "satu nilai author" yang dipecah jadi dua), melainkan **dua fakta independen** yang kebetulan sama-sama menjelaskan siapa penulisnya, sehingga keputusan akhir `meta_author` dan `meta_maintainer` dimodelkan sebagai **simple attribute terpisah** (`meta_author`, `meta_author_email`, `meta_author_email_verified` & `meta_maintainer`, `maintainer_email`, `maintainer_email_verified`), bukan composite attribute.
2. Karena syarat yang sama juga berlaku buat `yanked` dan `yanked_reason`, `yanked` (boolean) dan `yanked_reason` (teks) bukan dua pecahan dari satu nilai yang sama, melainkan pola **flag/status DAN detail kondisional** yang cuma terisi kalau flag-nya `true`, sehingga keputusan akhir keduanya tetap dimodelkan sebagai **simple attribute terpisah**, bukan composite attribute.

### RELEASE_FILE (_weak entity_, owner: RELEASE, discriminator: `filename`)

**Merepresentasikan apa?** Satu file distribusi yang bisa di-download dari sebuah release (source distribution atau wheel).

| Atribut | Klasifikasi | Bukti render |
|---|---|---|
| `filename` | discriminator | daftar file + judul File details |
| `path` | simple | href "Download URL" di File details |
| `packagetype()` | derived | diproses dari ekstensi `filename` (`.whl` = wheel, selain itu = sdist) |
| `size` | simple | daftar file + File details "Size:" |
| `upload_time` | simple | File details "Upload date:" |
| `uploaded_via` | simple, nullable | File details "Uploaded via:" |
| `is_trusted_publishing` | simple | File details "Uploaded using Trusted Publishing? Yes/No" |
| `{wheel_tag}` | multivalued | File details "Tags:" |

### FILE_HASH (_weak entity_, owner: RELEASE_FILE, discriminator: `algorithm`)

**Merepresentasikan apa?** Satu nilai checksum file untuk verifikasi integritas download.

| Atribut | Klasifikasi | Bukti render |
|---|---|---|
| `algorithm` | discriminator | kolom Algorithm (SHA256, MD5, BLAKE2b-256) |
| `digest` | simple | kolom Hash digest |

### PROJECT_LINK (_weak entity_, owner: RELEASE, discriminator: `label`)

**Merepresentasikan apa?** Satu link eksternal yang dicantumkan pemilik package (Homepage, Documentation, Source, dst).

| Atribut | Klasifikasi | Bukti render |
|---|---|---|
| `label` | discriminator | nama link |
| `url` | simple | href link |
| `verified` | simple | Section tempat link muncul (Verified vs Unverified details) |

### MAINTAINER (_strong entity_)

**Merepresentasikan apa?** Satu akun PyPI yang terhubung ke suatu package/project.

| Atribut | Klasifikasi | Bukti render |
|---|---|---|
| `username` | key, simple | teks di samping avatar, juga bagian URL profil `pypi.org/user/<username>/` |
| `joined_at` | simple, nullable | halaman profil user, "Joined ..." (WAJIB akses `pypi.org/user/<username>/`); NULL untuk akun yang tidak menampilkan metadiv "Date joined" sama sekali (mis. akun organisasi seperti `aws`, atau sebagian akun individual lama) |
| `maintained_project_count()` | derived | dihitung dari COUNT relationship `maintained_by` |

### CLASSIFIER (_strong entity_)

**Merepresentasikan apa?** Satu tag Trove classifier standar (kategori terstruktur PyPI), dishare oleh banyak package.

| Atribut | Klasifikasi | Bukti render |
|---|---|---|
| `category` | key (composite), simple | heading grup (misalnya "Programming Language") |
| `value` | key (composite), simple | item di bawah grup (misalnya "Python :: 3.13") |

### ATTESTATION (_strong entity_)

**Merepresentasikan apa?** Satu bukti provenance kriptografis (PEP 740 / Sigstore) bahwa sebuah file dibangun dan dipublish otomatis lewat CI/CD tertentu. Satu file bisa punya banyak attestation.

| Atribut | Klasifikasi | Bukti render |
|---|---|---|
| `sigstore_log_index` | key, simple | "Sigstore transparency entry:", unik secara global di transparency log Sigstore |
| `statement_type` | simple | "Statement type:" |
| `predicate_type` | simple | "Predicate type:" |
| `subject_name` | simple | "Subject name:" |
| `subject_digest` | simple | "Subject digest:" |
| `integration_time` | simple | "Sigstore integration time:" |
| `source_repo` | simple, nullable | "Source repository / Permalink" |
| `source_reference` | simple, nullable | "Branch / Tag:" |
| `token_issuer` | simple | "Token Issuer:" |
| `runner_environment` | simple, nullable | "Runner Environment:" |
| `publisher_workflow` | simple, nullable | "Publication workflow:" |
| `trigger_event` | simple, nullable | "Trigger Event:" (live packaging: "workflow_dispatch") |

**CATATAN:**
1. Karena aturannya weak entity itu harus *existence dependent* terhadap owner DAN gak punya key sendiri yang independen (sifat AND), dan ATTESTATION cuma memenuhi syarat pertama (*existence dependent* terhadap RELEASE_FILE) tapi tetap punya key natural sendiri (`sigstore_log_index`, diterbitkan server **Rekor**, transparency log milik Sigstore, unik secara global, gak bentrok antar file maupun antar package), sehingga keputusan akhir dia dimodelkan sebagai **strong entity dengan identifier `sigstore_log_index`**.

## Relationship Set

| Relationship | Antara | Kardinalitas | Identifying? | Partisipasi |
|---|---|---|---|---|
| `has` | PACKAGE — RELEASE | 1:N | Ya | RELEASE total , PACKAGE parsial |
| `owned_by` | PACKAGE — ORGANIZATION | N:1 | Tidak | Dua-duanya parsial |
| `distributes` | RELEASE — RELEASE_FILE | 1:N | Ya | RELEASE_FILE total, RELEASE parsial |
| `checks` | RELEASE_FILE — FILE_HASH | 1:N (tepat 3) | Ya | FILE_HASH total, RELEASE_FILE parsial |
| `links` | RELEASE — PROJECT_LINK | 1:N | Ya | PROJECT_LINK total, RELEASE parsial |
| `proves` | RELEASE_FILE — ATTESTATION | 1:N | Tidak | ATTESTATION total, RELEASE_FILE parsial |
| `maintained_by` | PACKAGE — MAINTAINER | M:N | Tidak | Dua-duanya parsial |
| `tagged_with` | RELEASE — CLASSIFIER | M:N | Tidak | Dua-duanya parsial |

**CATATAN:**
1. `checks`, "tepat 3" adalah business constraint di luar notasi kardinalitas standar, di-hardcode sebagai tiga baris SHA256/MD5/BLAKE2b-256.