# `003_data_transformer`

Tahap pembersihan dan validasi data hasil scraping [`001_data_scraping`](../001_data_scraping/README.md), sebelum data dimuat ke database oleh [`004_data_loader`](../004_data_loader/README.md).

## Apa itu tahap ini

`003_data_transformer` murni membersihkan dan memvalidasi nilai. Tahap ini tidak membuat UUID dan tidak menyelesaikan foreign key, keduanya jadi tanggung jawab [`004_data_loader`](../004_data_loader/README.md) saat proses load ke Postgres. Bentuk data (satu file JSON per entity, dengan natural key seperti nama package dan versi) tetap sama persis dengan output [`001_data_scraping`](../001_data_scraping/README.md). Yang berubah hanya isi nilainya: sesuai schema, tipe benar, panjang sesuai batas kolom, dan konsisten untuk satu relasi yang bisa rusak saat scraping, yaitu `maintained_by` ke `maintainer`.

Batasan ini sengaja dijaga supaya `003` tidak perlu tahu apa-apa soal Postgres, UUID, atau urutan INSERT, dan `004` tidak perlu menebak-nebak apakah nilai yang mau di-load sudah valid.

## Setup awal (virtual environment)

Dari dalam `003_01_src/`, buat virtual environment sekali saja:

```bash
python -m venv .venv
```

Cara masuk (aktivasi) `.venv` berbeda tergantung OS dan shell yang dipakai:

| OS | Shell | Perintah aktivasi |
|---|---|---|
| Windows | PowerShell | `.venv\Scripts\Activate.ps1` |
| Windows | Command Prompt (cmd) | `.venv\Scripts\activate.bat` |
| macOS | zsh / bash | `source .venv/bin/activate` |
| Ubuntu / Linux | bash | `source .venv/bin/activate` |

Prompt terminal akan menampilkan `(.venv)` di depan kalau aktivasi berhasil. Untuk keluar dari virtual environment, jalankan `deactivate` di semua platform.

## Cara menjalankan

Dari dalam `003_01_src/`, dengan **virtual environment aktif** (lihat langkah di atas):

```bash
pip install -r requirements.txt
python main.py
```

Tidak ada argumen. Program otomatis mencari folder input dan menulis output ke lokasi tetap.

## Sumber data input

Input diambil dari folder `pxs_<timestamp>` terbaru di [`001_data_scraping/001_02_data/`](../001_data_scraping/001_02_data/). Deteksi dilakukan otomatis: semua folder `pxs_*` diurutkan secara leksikografis, lalu yang namanya terbesar dipilih (format timestamp ISO 8601 basic memang urut benar sebagai string, jadi tidak perlu parsing tanggal). Kalau tidak ada folder `pxs_*` sama sekali, atau folder yang terpilih tidak punya salah satu file entity yang diharapkan, program berhenti dengan error dan tidak menulis output apa pun.

Output ditulis ke `003_02_data_cleaned/pxs_<timestamp>/`, memakai timestamp yang sama persis dengan folder input, supaya gampang menelusuri hasil cleaning mana yang berasal dari scrape mentah mana.

## Aturan pembersihan

Aturan cleaning didorong oleh registry kolom di [`schema.py`](003_01_src/schema.py), bukan ditulis manual per entity. Setiap kolom pada tiap entity punya spesifikasi: boleh null atau tidak, batas panjang (kalau ada), klasifikasi `identifier` atau `descriptive`, dan daftar enum yang valid (kalau kolom itu enum). [`clean.py`](003_01_src/clean.py) menjalankan aturan yang sama ke semua entity berdasarkan registry ini.

- **Kolom wajib diisi (`nullable=False`) yang kosong**: baris di-drop. Kolom yang memang boleh kosong tapi hilang dari baris sumber diisi `null` eksplisit, tidak dibiarkan hilang begitu saja, supaya konsumen data hilir tidak perlu menebak apakah suatu key yang tidak ada berarti null atau berarti baris itu cacat.
- **Kolom `identifier` yang kepanjangan**: barisnya di-drop, bukan dipotong. Contoh: `maintainer.username`, `classifier.category`, `classifier.value`, `file_hash.digest`, `release_extra.extra_name`, dan kolom-kolom VARCHAR di `attestation`. Kolom jenis ini berisi URI, hash, nama file, atau identitas lain yang kalau dipotong bisa berubah makna atau malah bertabrakan dengan identitas lain, jadi lebih aman dibuang daripada dipotong diam-diam.
- **Kolom `descriptive` yang kepanjangan**: dipotong jadi `n-3` karakter lalu ditambah `"..."`, bukan di-drop. Contoh: `organization.display_name`, `release.summary`, `release_file.uploaded_via`, `release_keyword.keyword`, `release_file_tag.wheel_tag`. Kolom jenis ini teks bebas, jadi versi terpotong dengan elipsis masih bermakna dan tidak menyesatkan.
- **Validasi enum**: kalau nilainya tidak termasuk daftar yang diperbolehkan, baris di-drop. Berlaku untuk `lifecycle_status` (`archived`, `deprecated`, `quarantined`, atau null) di `package` dan `release`, serta `algorithm` (`SHA256`, `MD5`, `BLAKE2b-256`) di `file_hash`.
- **Kolom tanpa batas panjang** (misalnya `license`, `description`, `url`, `path`) diteruskan apa adanya karena memang tidak ada yang perlu divalidasi.

Setiap baris yang di-drop atau dipotong dicatat dan ditampilkan langsung ke user selama proses berjalan, sehingga baris bermasalah bisa ditelusuri satu per satu, bukan cuma angka total di akhir.

## Backfill placeholder maintainer

Saat scraping, profil sebagian maintainer bisa gagal diambil karena kena bot-detection. Akibatnya, ada baris `maintained_by` yang menunjuk ke `maintainer_username` yang tidak punya baris di `maintainer.json`. Karena foreign key `maintained_by` ke `maintainer` di Postgres bersifat `NOT NULL`, referensi yang menggantung begini akan gagal saat load kalau dibiarkan.

Setelah cleaning per entity selesai, `003` memindai `maintained_by` untuk mencari username yang belum punya baris `maintainer` yang cocok, lalu membuat baris placeholder untuk masing-masing: `{"username": "<nama>", "joined_at": null}`.

`joined_at` sengaja diisi `null`, bukan tanggal sentinel seperti `1970-01-01`. Null di sini secara semantik berarti "tidak diketahui", yang memang sesuai kondisi sebenarnya, profil maintainer itu belum pernah berhasil diambil sama sekali. Tanggal sentinel justru menyiratkan ada tanggal pasti yang diketahui, padahal tidak. Perlakuan ini juga konsisten dengan maintainer lain yang memang tidak punya tanggal "Date joined" di profil PyPI-nya.

## Struktur folder

```
003_data_transformer/
├── 003_01_src/
│   ├── .venv/              (virtualenv, tidak masuk git)
│   ├── requirements.txt    (pandas, pytest)
│   ├── main.py             # entrypoint: cari run terbaru, baca -> bersihkan -> backfill -> tulis -> laporan
│   ├── schema.py           # registry kolom per entity (data murni, tanpa logika)
│   └── clean.py            # mesin cleaning generik berbasis schema, tanpa I/O
└── 003_02_data_cleaned/
    └── pxs_<timestamp>/    (output, satu run dalam satu waktu)
```

Pembagian tanggung jawabnya:

- [`schema.py`](003_01_src/schema.py) cuma data, tidak ada logika. Mendaftarkan kolom apa saja yang ada per entity, dan aturan apa yang berlaku untuk masing-masing.
- [`clean.py`](003_01_src/clean.py) mesin cleaning, berupa fungsi murni yang beroperasi di atas baris data (list of dict), tidak menyentuh file sama sekali. Satu implementasi dipakai untuk semua entity karena aturannya datang dari `schema.py`, bukan ditulis ulang per entity.
- [`main.py`](003_01_src/main.py) yang pegang I/O: menemukan folder input, membaca dan menulis file JSON, mengorkestrasi urutan cleaning per entity lalu backfill maintainer, dan mencetak laporan progres ke user.
