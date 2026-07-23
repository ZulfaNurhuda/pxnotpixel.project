# Pengecekan Functional Dependencies, Candidate Key, dan Bentuk Normal untuk PyPI Extraction Data

## RELASI: organization

**Functional Dependency**
```
F = { organization_id -> display_name, name }
```

**Candidate Key**: `organization_id`

**Bentuk Normal**
Satu-satunya FD non-trivial mempunyai LHS `organization_id` yang merupakan superkey. Tidak ada FD lain dengan LHS yang bukan superkey. Relasi organization berada pada BCNF (otomatis juga 3NF, 2NF).

## RELASI: maintainer

**Functional Dependency**
```
F = { maintainer_id -> joined_at, username }
```

**Candidate Key**: `maintainer_id`

**Bentuk Normal**
Satu-satunya FD non-trivial mempunyai LHS `maintainer_id` yang superkey. Relasi maintainer berada pada BCNF.

## RELASI: classifier

**Functional Dependency**
```
F = { classifier_id -> category, value; category, value -> classifier_id }
```

**Super Key**: `classifier_id`, `(category, value)`

**Candidate Key**: `classifier_id`

**Bentuk Normal**
Kedua FD non-trivial punya LHS yang superkey (`classifier_id` atau pasangan `category`-`value`). Tidak ada FD dengan LHS bukan superkey (category sendirian atau value sendirian tidak menentukan atribut lain). Relasi classifier berada pada BCNF.

## RELASI: package

**Functional Dependency**
```
F = { package_id -> organization_id, lifecycle_status, name; name -> package_id, organization_id, lifecycle_status }
```

**Super Key**: `package_id`, `name`

**Candidate Key**: `package_id`

**Bentuk Normal**
Seluruh FD non-trivial punya LHS superkey (`package_id` atau `name`). Tidak ditemukan FD tersembunyi seperti `organization_id` yang menentukan `lifecycle_status`. Relasi package berada pada BCNF.

## RELASI: release

**Functional Dependency**
```
F = { release_id -> package_id, created, is_prerelease, yanked, lifecycle_status, version, yanked_reason, summary, license, requires_python; package_id, version -> release_id, created, is_prerelease, yanked, lifecycle_status, yanked_reason, summary, license, requires_python }
```

**Super Key**: `release_id`, `(package_id, version)`

**Candidate Key**: `release_id`

**Bentuk Normal**
Seluruh FD non-trivial punya LHS superkey. Relasi kondisional antara `yanked` dan `yanked_reason` (`yanked_reason` cuma relevan kalau `yanked` bernilai true) bukan functional dependency, karena `yanked` tidak menentukan isi `yanked_reason` (dua release yang sama-sama yanked bisa punya alasan berbeda), melainkan integrity constraint di luar cakupan analisis FD. Relasi release berada pada BCNF.

## RELASI: release_detail

**Functional Dependency**
```
F = { release_id -> meta_author_email_verified, meta_maintainer_email_verified, description, meta_author, meta_author_email, meta_maintainer, meta_maintainer_email }
```

**Candidate Key**: `release_id`

**Bentuk Normal**
Satu-satunya FD non-trivial punya LHS `release_id` yang superkey. `meta_author_email` tidak menentukan `meta_author_email_verified` (dua baris bisa punya email yang sama tapi status verifikasi berbeda). Relasi release_detail berada pada BCNF.

## RELASI: release_file

**Functional Dependency**
```
F = { release_file_id -> release_id, size, upload_time, is_trusted_publishing, filename, path, uploaded_via; release_id, filename -> release_file_id, size, upload_time, is_trusted_publishing, path, uploaded_via }
```

**Super Key**: `release_file_id`, `(release_id, filename)`

**Candidate Key**: `release_file_id`

**Bentuk Normal**
Seluruh FD non-trivial punya LHS superkey. Relasi release_file berada pada BCNF.

## RELASI: file_hash

**Functional Dependency**
```
F = { release_file_id, algorithm -> digest }
```

**Candidate Key**: `(release_file_id, algorithm)`

**Bentuk Normal**
Satu-satunya FD non-trivial punya LHS `(release_file_id, algorithm)` yang superkey, dan `digest` bergantung ke KESELURUHAN pasangan itu (bukan partial dependency ke salah satu bagian saja, karena `digest` berbeda untuk `algorithm` berbeda meski `release_file_id` sama). Relasi file_hash berada pada BCNF.

## RELASI: project_link

**Functional Dependency**
```
F = { release_id, label -> verified, url }
```

**Candidate Key**: `(release_id, label)`

**Bentuk Normal**
Satu-satunya FD non-trivial punya LHS `(release_id, label)` yang superkey, dan `verified` serta `url` bergantung ke keseluruhan pasangan itu, bukan partial dependency (dua release berbeda bisa sama-sama punya `label` "Homepage" dengan `url` berbeda). Relasi project_link berada pada BCNF.

## RELASI: maintained_by

**Functional Dependency**
```
F = { }
```
_(tidak ada atribut non-key, sehingga tidak ada FD non-trivial yang mungkin terbentuk)_

**Candidate Key**: `(package_id, maintainer_id)`

**Bentuk Normal**
Tidak ada atribut non-key sama sekali di relasi ini, sehingga tidak mungkin ada pelanggaran bentuk normal manapun. Relasi maintained_by berada pada BCNF secara trivial.

## RELASI: tagged_with

**Functional Dependency**
```
F = { }
```
_(tidak ada atribut non-key, sehingga tidak ada FD non-trivial yang mungkin terbentuk)_

**Candidate Key**: `(release_id, classifier_id)`

**Bentuk Normal**
Tidak ada atribut non-key. Relasi tagged_with berada pada BCNF secara trivial.

## RELASI: release_keyword

**Functional Dependency**
```
F = { }
```
_(tidak ada atribut non-key, sehingga tidak ada FD non-trivial yang mungkin terbentuk)_

**Candidate Key**: (release_id, keyword)

**Bentuk Normal**
Tidak ada atribut non-key. Relasi release_keyword berada pada BCNF secara trivial.

## RELASI: release_extra

**Functional Dependency**
```
F = { }
```
_(tidak ada atribut non-key, sehingga tidak ada FD non-trivial yang mungkin terbentuk)_

**Candidate Key**: `(release_id, extra_name)`

**Bentuk Normal**
Tidak ada atribut non-key. Relasi release_extra berada pada BCNF secara trivial.

## RELASI: release_file_tag

**Functional Dependency**
```
F = { }
```
_(tidak ada atribut non-key, sehingga tidak ada FD non-trivial yang mungkin terbentuk)_

**Candidate Key**: `(release_file_id, wheel_tag)`

**Bentuk Normal**
Tidak ada atribut non-key. Relasi release_file_tag berada pada BCNF secara trivial.

## RELASI: attestation

**Functional Dependency**
```
F = { sigstore_log_index -> release_file_id, integration_time, statement_type, predicate_type, subject_name, subject_digest, source_repo, source_reference, token_issuer, runner_environment, publisher_workflow, trigger_event }
```

**Candidate Key**: `sigstore_log_index`

**Bentuk Normal**
Satu-satunya FD non-trivial punya LHS sigstore_log_index yang superkey. Relasi attestation berada pada BCNF.
