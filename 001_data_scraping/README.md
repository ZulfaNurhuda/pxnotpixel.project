# `001_data_scraping`

Subsistem pertama pipeline `pxnotpixel`: scraper statis yang mengambil data 100 package PyPI langsung dari `pypi.org`, lalu menuliskannya sebagai JSON per entity. Programnya bernama **`pxs`** (PyPI eXtractor Script).

## Apa itu `pxs`

`pxs` adalah scraper statis. Halaman `pypi.org` di-render penuh di server (Warehouse, aplikasi PyPI resmi), jadi seluruh data yang dibutuhkan sudah ada di HTML respons pertama tanpa perlu headless browser atau eksekusi JavaScript. `pxs` cukup melakukan HTTP GET biasa lalu parse HTML dengan CSS selector.

Program ini dijalankan langsung tanpa argumen atau flag apapun. Sekali jalan, `pxs` mengambil data 100 package (nama, rilis, file, hash, maintainer, organisasi, classifier, dan seterusnya) dan menulis hasilnya sebagai kumpulan file JSON di direktori output baru.

## Cara menjalankan

### 1. Bootstrap daftar package (satu kali)

`pxs` butuh daftar nama package sebagai kandidat untuk di-scrape, disimpan di [`001_01_src/pxs/meta/top_pypi.txt`](001_01_src/pxs/meta/top_pypi.txt) (satu nama package per baris, terurut menurun berdasarkan jumlah download). File ini dihasilkan dari sumber eksternal ([hugovk.dev/top-pypi-packages](https://hugovk.dev/top-pypi-packages/)) lewat tindakan setup satu kali, bukan bagian dari alur `pxs` yang jalan berulang. File ini sudah tersedia di repo, bootstrap ulang hanya perlu dilakukan kalau daftarnya ingin di-refresh.

### 2. Jalankan scraper

```bash
cd 001_data_scraping/001_01_src/pxs
go run .
```

Tidak ada argumen atau flag. Program membaca `meta/top_pypi.txt` relatif terhadap direktori kerja saat itu, jadi perintah di atas harus dijalankan dari dalam `001_01_src/pxs/`.

Selama berjalan, `pxs` mencetak progres per package ke stdout, dan pesan lewati (skip) ke stderr kalau ada package/versi/profil maintainer yang gagal diambil. Di akhir, `pxs` mencetak ringkasan jumlah package yang berhasil diambil.

## Sumber data

Dua jenis halaman `pypi.org` yang di-fetch:

- `pypi.org/project/<nama>/`: halaman utama package, dan `pypi.org/project/<nama>/<versi>/` untuk tiap versi historis yang di-scrape. Kedua bentuk halaman ini punya struktur HTML yang sama, cuma beda versi yang ditampilkan.
- `pypi.org/user/<username>/`: profil maintainer, satu-satunya sumber untuk field `joined_at` (tidak ada di halaman project).

Catatan penting soal halaman utama tanpa versi (`pypi.org/project/<nama>/`): halaman ini menampilkan **rilis stabil terbaru**, bukan otomatis entri paling atas di tab "Release history". Kalau rilis terbaru dalam riwayat adalah pre-release (contoh nyata yang ditemukan: `pydantic` punya `2.14.0a1` di riwayat rilis padahal halaman utama menunjukkan `2.13.4` sebagai versi stabil), keduanya berbeda. Karena itu `pxs` **tidak** mengasumsikan versi dari entri pertama riwayat rilis, melainkan membaca versi asli langsung dari header halaman (`.package-header__name`) lewat `parse.CurrentVersion`. Entri pertama riwayat rilis hanya dipakai sebagai fallback kalau header gagal diparsing.

Setiap request HTTP wajib membawa header `User-Agent` (`pxs/<versi> (...)`), sesuai kebijakan penggunaan wajar pypi.org.

## Peta selector HTML kunci

Ringkasan selector CSS per entity. Dokumen selengkapnya (per field, termasuk pengecualian dan edge case) ada di [`BRIEF.md`](../BRIEF.md) bagian "Peta sumber HTML per field".

| Entity / field | Selector kunci |
|---|---|
| Nama & versi package | `.package-header__name` |
| Status lifecycle (quarantined) | `.package-header__right .status-badge span` |
| Owner organisasi | `.sidebar-section.verified`, cari `<h6>Owner</h6>` lalu `ul.vertical-tabs__list li a` sibling |
| Tanggal rilis | `.package-header__date time[datetime]` |
| Ringkasan & deskripsi | `.package-description__summary`, `.project-description` |
| License, Requires-Python, Author, Maintainer, keyword, provides-extra | `.sidebar-section.verified` / `.sidebar-section.unverified`, section `<h6>Meta</h6>`, dicocokkan lewat label `<strong>` |
| Riwayat rilis (pre-release, yanked) | `.release-timeline .release`, badge `.badge--warning` / `.badge--danger` di dalam `.release__version` |
| Daftar & detail file | `#files` (klasifikasi sdist/wheel dari heading `<h3>`), lalu `div[id=<filename>]` untuk detail per file |
| Hash file | `table.table--hashes tbody tr` |
| Project links | `.sidebar-section.verified` / `.unverified`, `<h6>Project links</h6>` lalu `ul.vertical-tabs__list li a` |
| Classifier | `.sidebar-section__classifiers li strong` (kategori), `li ul li a` (value) |
| Maintainer (username) | `.sidebar-section.verified`, `<h6>Maintainers</h6>`, `.sidebar-section__user-gravatar-text` |
| Attestation / provenance | Section "Provenance" di dalam detail file, `<li>` polos dicocokkan lewat prefix teks label |
| `joined_at` maintainer | Halaman profil terpisah, `.author-profile__metadiv` dengan ikon `fa-calendar-alt`, `<time datetime>` |

Catatan penting: pypi.org merender sebagian section sidebar (`.sidebar-section.verified` dan `.sidebar-section.unverified`) **dua kali** di HTML statis, satu salinan yang terlihat dan satu salinan tersembunyi untuk widget tab responsif di layar sempit. Tanpa penanganan khusus, tiap baris data di section itu akan terekstrak dua kali lipat. `pxs` menghindari ini dengan selalu mengambil salinan pertama saja (`.First()` pada hasil `doc.Find(...)`, lihat `verifiedSection` dan `unverifiedSection` di [`internal/parse/parse.go`](001_01_src/pxs/internal/parse/parse.go)).

Tanggal/waktu selalu diambil dari atribut `datetime` pada elemen `<time>` (format ISO 8601), bukan dari teks tampilan elemen tersebut. Teks tampilan itu dilokalisasi di sisi client lewat JavaScript dan bisa berbeda hari kalender dari instant UTC aslinya.

## Alur data

```
meta/top_pypi.txt
        |
        v
   [ fetch ]  --  HTTP GET dengan header User-Agent wajib, timeout terkonfigurasi
        |
        v
   [ parse ]  --  HTML -> struct entity per goquery selector
        |
        v
   [ writer ] --  JSON per entity, dedup entity master (organization, maintainer, classifier)
        |
        v
001_02_data/pxs_<timestamp>/*.json
```

1. **Fetch**: [`internal/fetch`](001_01_src/pxs/internal/fetch/fetch.go) melakukan HTTP GET biasa, menyisipkan header `User-Agent`, dan menegakkan timeout. Tidak menangani jeda antar request (itu tanggung jawab loop utama).
2. **Parse**: [`internal/parse`](001_01_src/pxs/internal/parse/parse.go) menerima byte HTML mentah dan mengembalikan struct Go per entity, murni tanpa I/O.
3. **Writer**: [`internal/writer`](001_01_src/pxs/internal/writer/writer.go) menulis tiap baris entity ke file JSON yang sesuai, dan melakukan dedup untuk tiga entity master/shared di seluruh run (lihat bagian Kebijakan).
4. **Output**: seluruh file JSON diletakkan di `001_data_scraping/001_02_data/pxs_<timestamp>/`, dengan `<timestamp>` diambil saat program mulai jalan, supaya tiap eksekusi punya folder tersendiri dan tidak menimpa hasil eksekusi sebelumnya.

## Kebijakan penanganan error

- **Tanpa retry.** Satu fetch yang gagal (status non-2xx atau error jaringan) langsung dianggap gagal untuk unit kerja itu, tidak dicoba ulang.
- **Skip-and-continue per unit kerja**, bukan gagal total satu kali error:
  - Package yang gagal di-fetch pada halaman utamanya dilewati, digantikan kandidat berikutnya dari daftar `top_pypi.txt` (backfill), supaya total tetap mencapai 100 package selama daftar kandidat masih cukup.
  - Versi historis yang gagal di-fetch atau di-parse hanya melewatkan baris versi itu, package tetap lanjut diproses dengan versi-versi lain yang berhasil.
  - Profil maintainer yang gagal di-fetch atau di-parse hanya melewatkan baris `joined_at` maintainer itu (jadi kosong), tidak menggagalkan package.
- **Field yang memang tidak ada bukan error.** Contoh: sebagian akun maintainer (mis. akun organisasi) tidak menampilkan tanggal bergabung sama sekali di profilnya, hasilnya kosong, bukan dianggap kegagalan fetch/parse.
- **Jeda antar package**: minimal 5 detik sebelum fetch halaman utama package berikutnya. Fetch versi historis dan fetch profil maintainer tidak punya jeda tambahan.
- **Cap riwayat rilis**: maksimum 10 versi terbaru per package yang diambil detailnya (versi terbaru dari halaman utama, sisanya dari halaman per-versi).
- Data yang sudah berhasil dikumpulkan tetap ditulis ke output meski program berhenti karena error fatal di tengah jalan.

## Struktur folder

```
001_data_scraping/
├── 001_01_src/
│   └── pxs/
│       ├── go.mod, go.sum
│       ├── main.go              orkestrasi: baca daftar package, loop scrape, finalisasi output
│       ├── meta/
│       │   └── top_pypi.txt     hasil bootstrap, daftar kandidat package
│       └── internal/
│           ├── config/          konstanta terpusat (jumlah package, cap rilis, delay, timeout, dll)
│           ├── fetch/           HTTP client dengan User-Agent wajib
│           ├── parse/           HTML -> struct entity via goquery
│           └── writer/          tulis entity ke JSON per file, dedup entity master
└── 001_02_data/
    └── pxs_<timestamp>/         output tiap eksekusi, satu folder per run
```

## File JSON entity yang dihasilkan

Setiap file berisi satu array JSON, satu elemen per baris data. Skema kolom lengkap ada di [`002_data_modelling`](../002_data_modelling/README.md) ([`px_RELATIONAL_DETAILS.md`](../002_data_modelling/002_02_relational/px_RELATIONAL_DETAILS.md)). Daftar di bawah ini cuma ringkasan field utama. Karena `pxs` jalan sebelum database ada, seluruh relasi antar entity memakai natural key (nama package, versi, username) dan bukan UUID surrogate. Resolusi ke foreign key surrogate jadi tanggung jawab `004_data_loader`.

| File | Field utama | Catatan |
|---|---|---|
| `package.json` | `name`, `lifecycle_status`, `organization_owner` | satu baris per package |
| `organization.json` | `name`, `display_name` | entity master, dedup per nama |
| `maintainer.json` | `username`, `joined_at` | entity master, dedup per username |
| `maintained_by.json` | `package_name`, `maintainer_username` | relasi package - maintainer |
| `release.json` | `package_name`, `version`, `created`, `is_prerelease`, `yanked`, `lifecycle_status`, `yanked_reason`, `summary`, `license`, `requires_python` | natural key `package_name + version` |
| `release_detail.json` | `package_name`, `version`, `description`, `meta_author(_email)(_verified)`, `meta_maintainer(_email)(_verified)` | dipisah dari `release.json` (partisi vertikal, kolom besar seperti deskripsi HTML) |
| `release_file.json` | `package_name`, `version`, `filename`, `path`, `size`, `upload_time`, `is_trusted_publishing`, `uploaded_via`, `packagetype` | `packagetype` = `sdist` atau `wheel` |
| `file_hash.json` | `package_name`, `version`, `filename`, `algorithm`, `digest` | selalu 3 baris per file (SHA256, MD5, BLAKE2b-256) |
| `project_link.json` | `package_name`, `version`, `label`, `url`, `verified` | `verified` menandai sumber section (verified/unverified) |
| `classifier.json` | `category`, `value` | entity master, dedup per pasangan kategori+value |
| `tagged_with.json` | `package_name`, `version`, `category`, `value` | relasi release - classifier |
| `release_keyword.json` | `package_name`, `version`, `keyword` | satu baris per keyword |
| `release_extra.json` | `package_name`, `version`, `extra_name` | satu baris per provides-extra |
| `release_file_tag.json` | `package_name`, `version`, `filename`, `wheel_tag` | hanya untuk file wheel |
| `attestation.json` | `package_name`, `version`, `filename`, `sigstore_log_index`, `integration_time`, `statement_type`, `predicate_type`, `subject_name`, `subject_digest`, `source_repo`, `source_reference`, `token_issuer`, `runner_environment`, `publisher_workflow`, `trigger_event` | hanya ada kalau file punya provenance (Sigstore) |

Tiga entity di atas (`organization`, `maintainer`, `classifier`) berperan sebagai tabel master/shared lintas package dalam satu kali eksekusi, jadi `writer` melakukan dedup berbasis natural key (nama organisasi, username, pasangan kategori+value) supaya tidak ada baris duplikat di file JSON-nya walau data yang sama muncul di banyak package.
