-- #1 Query untuk mendapatkan data 10 package dengan jumlah release terbanyak beserta nama organisasi pemiliknya
SELECT DISTINCT
    p.package_id,
    p.name,
    o.display_name,
    (
        SELECT COUNT(*) 
        FROM release r 
        WHERE r.package_id = p.package_id
    ) AS release_count
FROM package p, organization o
WHERE p.org_id = o.organization_id
ORDER BY (
    SELECT COUNT(*) 
    FROM release r 
    WHERE r.package_id = p.package_id
) DESC
LIMIT 10;

-- #2 Query untuk mendapatkan data release dari package yang memiliki maintainer dengan username mengandung 'lib'
SELECT
    r.release_id,
    r.version,
    p.name,
    (
        SELECT COUNT(*) 
        FROM maintained_by mb 
        WHERE mb.package_id = p.package_id
    ) AS maintainer_count
FROM release r, package p
WHERE r.package_id = p.package_id
AND p.package_id IN (
    SELECT mb.package_id
    FROM maintained_by mb, maintainer m
    WHERE mb.maintainer_id = m.maintainer_id
      AND m.username LIKE '%lib%'
)
ORDER BY p.name;

-- #3 Query untuk mendapatkan data package yang belum pernah dimiliki oleh maintainer manapun
SELECT p.package_id, p.name
FROM package p
WHERE p.package_id NOT IN (
    SELECT package_id 
    FROM maintained_by
)
ORDER BY p.name;

-- #4 Query untuk mendapatkan data release_file beserta hash SHA256 dari package tertentu (pencarian nama case-insensitive)
SELECT *
FROM release_file rf, release r, package p, file_hash fh
WHERE rf.release_id = r.release_id
  AND r.package_id = p.package_id
  AND fh.release_file_id = rf.release_file_id
  AND UPPER(p.name) = UPPER('numpy')
  AND fh.algorithm = 'SHA256';

-- #5 Query untuk mendapatkan data ringkasan jumlah release per organisasi, diurutkan dari yang terbanyak
SELECT
    o.organization_id,
    o.display_name,
    (
        SELECT COUNT(*)
        FROM release r
        WHERE r.package_id IN (
            SELECT p.package_id
            FROM package p
            WHERE p.org_id = o.organization_id
        )
    ) AS total_release
FROM organization o
WHERE o.organization_id IN (
    SELECT p.org_id 
    FROM package p 
    WHERE p.org_id IS NOT NULL
)
ORDER BY total_release DESC;
