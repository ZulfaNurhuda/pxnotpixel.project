# `004_data_loader`

Tahap akhir ETL: memuat data hasil pembersihan [`003_data_transformer`](../003_data_transformer/README.md) ke database Postgres.

## Apa itu tahap ini

`004_data_loader` mengambil JSON natural-key hasil [`003_data_transformer`](../003_data_transformer/README.md) lalu memuatnya ke Postgres. Ini satu-satunya subsistem di pipeline yang bertanggung jawab membuat UUID surrogate dan menyelesaikan foreign key, [`001_data_scraping`](../001_data_scraping/README.md) dan `003_data_transformer` sengaja dijaga tetap natural-key-only supaya tidak perlu tahu apa-apa soal database.

Dua hal utama yang dikerjakan tahap ini:

- **Generate UUID surrogate**: setiap tabel di schema Postgres punya kolom primary key `DEFAULT uuidv7()`. `004` tidak membuat UUID sendiri di sisi Go, cukup `INSERT ... RETURNING` dan Postgres yang mengisi nilainya.
- **Resolusi natural key jadi foreign key**: baris input masih menunjuk ke entity lain lewat natural key (nama package, versi release, username maintainer, dan sejenisnya). `004` menyimpan peta natural-key ke UUID hasil insert di memori, lalu memakainya untuk mengisi kolom FK saat entity yang bergantung dimuat.

## Cara menjalankan

Dari dalam `004_01_src/`:

```bash
go run .
```

Tidak ada argumen. Program otomatis mencari folder input terbaru dan membaca kredensial database sendiri.

Syarat sebelum menjalankan: container `px_postgres` harus sudah jalan (lihat [`005_data_storing/README.md`](../005_data_storing/README.md)).

```bash
docker compose -f 005_01_docker/compose.yml up -d
```

Dijalankan dari [`005_data_storing/`](../005_data_storing/README.md).

## Sumber data input

Input diambil dari folder `pxs_<timestamp>` terbaru di [`003_data_transformer/003_02_data_cleaned/`](../003_data_transformer/003_02_data_cleaned/). Deteksi sama seperti `003`: semua folder `pxs_*` diurutkan leksikografis, folder dengan nama terbesar yang dipakai (timestamp ISO 8601 urut benar sebagai string, tidak perlu parsing tanggal).

## Kredensial database

Kredensial dibaca langsung dari `005_data_storing/005_01_docker/.env` saat runtime, lewat parse `KEY=VALUE` sederhana untuk baris `POSTGRES_PASSWORD`. `004` sengaja tidak punya file `.env` sendiri, satu sumber kebenaran untuk kredensial ini.

Nilai host, port, user, dan nama database tetap (bukan dari `.env`) karena memang konstan sesuai [`005_data_storing/005_01_docker/compose.yml`](../005_data_storing/005_01_docker/compose.yml):

| Parameter | Nilai |
|---|---|
| Host | `127.0.0.1` |
| Port | `5432` |
| User | `px_user` |
| Database | `px_db` |

## Urutan insert sesuai dependency FK

Ada 15 entity yang dimuat, urut dari yang tanpa dependency sampai yang paling bergantung. Resolusi natural key ke UUID dilakukan lewat peta di memori (`Resolver`), diisi begitu entity induknya selesai di-insert.

1. `organization`, `maintainer`, `classifier` - tidak punya dependency. Di-insert lebih dulu, ID hasilnya ditangkap lewat `RETURNING` dan disimpan di peta dengan key nama organisasi, username maintainer, dan pasangan (category, value).
2. `package` - menyelesaikan `organization_owner` jadi `org_id` (FK nullable, nama organisasi kosong berarti "tanpa organisasi", bukan referensi gagal) lewat peta organisasi. ID hasil disimpan dengan key nama package.
3. `release` - menyelesaikan nama package jadi `package_id`. ID hasil disimpan dengan key (nama package, versi).
4. `release_detail`, `release_file`, `project_link`, `release_keyword`, `release_extra` - menyelesaikan (nama package, versi) jadi `release_id`. `release_file` tambahan menangkap `release_file_id`, disimpan dengan key (nama package, versi, nama file).
5. `file_hash`, `release_file_tag`, `attestation` - menyelesaikan (nama package, versi, nama file) jadi `release_file_id`.
6. `maintained_by` - menyelesaikan nama package jadi `package_id` dan username maintainer jadi `maintainer_id`.
7. `tagged_with` - menyelesaikan (nama package, versi) jadi `release_id` dan (category, value) jadi `classifier_id`.

Urutan persis ini ada di `entityOrder` pada [`main.go`](004_01_src/main.go), dan urutan pemanggilan fungsi load di `loadAll`.

## Kebijakan TRUNCATE dan transaksi

- **TRUNCATE dulu, baru insert.** Bukan upsert. Semua 15 tabel di-TRUNCATE sekaligus lewat satu statement `TRUNCATE ... CASCADE` di awal transaksi, sebelum insert apa pun dimulai. `CASCADE` membuat urutan penyebutan tabel di statement TRUNCATE tidak relevan terlepas arah FK-nya. Setiap load dimulai dari tabel kosong, tidak ada logika upsert.
- **Satu transaksi untuk seluruh proses.** TRUNCATE sampai insert entity terakhir berjalan dalam satu transaksi database yang sama. Kalau seluruhnya berhasil, transaksi di-commit sekali di akhir.
- **Referensi natural key yang gagal diresolusi bersifat fatal.** Kalau sebuah baris menunjuk ke natural key yang tidak ditemukan di peta (misal `release_file` mengacu ke (nama package, versi) yang tidak punya baris `release`), proses berhenti dan transaksi di-rollback total, bukan baris itu dilewati. Pada tahap ini `003` sudah memvalidasi dan membersihkan data, jadi referensi yang gagal diresolusi menandakan bug nyata di pipeline, bukan kegagalan eksternal yang wajar seperti scrape yang diblokir. Ini beda dengan kebijakan skip-and-continue yang dipakai `001` dan `003` untuk kegagalan eksternal.
- **Tidak ada retry.** Ini operasi batch lokal terhadap database yang sudah jalan, tidak ada yang sifatnya transient untuk dicoba ulang.

## Strategi testing

Testing dibagi berdasarkan bagian mana yang sebenarnya berisiko salah:

- **Unit test (paket `testing` bawaan Go, tanpa database asli)**: menguji logika resolusi natural-key ke UUID, deteksi referensi yang hilang (harus menghasilkan error fatal, bukan diam-diam dilewati), dan urutan dependency. Semua ini fungsi murni yang beroperasi di atas slice entity dan peta di memori, tidak butuh I/O untuk diuji dengan benar.
- **Verifikasi manual terhadap database yang benar-benar jalan**: kebenaran SQL `INSERT`/`RETURNING`/`TRUNCATE` yang sesungguhnya diverifikasi dengan menjalankan `004` terhadap instance `px_postgres` docker-compose yang sudah jalan, lalu memeriksa jumlah baris dan contoh baris per tabel. Tidak memakai `testcontainers-go` atau dependency database sekali-pakai lain.

## Struktur folder

```
004_data_loader/004_01_src/
├── go.mod
├── go.sum
├── main.go                  # baca JSON, buka koneksi+transaksi, load berurutan, commit/rollback, laporan
└── internal/
    ├── config/
    │   └── config.go        # parameter koneksi DB, path .env, lokasi folder input
    ├── entity/
    │   └── entity.go        # satu struct Go per entity, tag JSON sama persis dengan output 003
    └── load/
        ├── load.go          # TRUNCATE + satu fungsi INSERT...RETURNING per entity
        └── resolve.go       # peta natural-key ke UUID (Resolver) dan lookup-nya
```

Pembagian tanggung jawabnya:

- [`config`](004_01_src/internal/config/config.go) menyimpan semua nilai koneksi database yang konstan, path `.env`, dan lokasi folder input, tidak ada angka ajaib tersebar di tempat lain.
- [`entity`](004_01_src/internal/entity/entity.go) berisi tipe data murni yang dipakai baik saat baca JSON (`main.go`) maupun saat load (`load.go`).
- `load` ([`load.go`](004_01_src/internal/load/load.go), [`resolve.go`](004_01_src/internal/load/resolve.go)) berisi TRUNCATE, satu fungsi insert per entity, dan peta resolusi natural-key ke UUID lewat `Resolver`.
- [`main.go`](004_01_src/main.go) mengorkestrasi urutan penuh: baca folder terbaru, baca kredensial, buka koneksi dan transaksi, TRUNCATE, load semua entity berurutan sesuai FK, commit atau rollback, lalu cetak laporan.
