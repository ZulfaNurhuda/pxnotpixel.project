-- #1 Query untuk mendapatkan data 10 package dengan jumlah release terbanyak beserta nama organisasi pemiliknya
SELECT
    p.package_id,
    p.name,
    o.display_name,
    rc.release_count
FROM package p
JOIN organization o ON o.organization_id = p.org_id
JOIN (
    SELECT package_id, COUNT(*) AS release_count
    FROM release
    GROUP BY package_id
) rc ON rc.package_id = p.package_id
ORDER BY rc.release_count DESC
LIMIT 10;

-- #2 Query untuk mendapatkan data release dari package yang memiliki maintainer dengan username mengandung 'lib'
SELECT
    r.release_id,
    r.version,
    p.name,
    mc.maintainer_count
FROM release r
JOIN package p ON p.package_id = r.package_id
JOIN (
    SELECT package_id, COUNT(*) AS maintainer_count
    FROM maintained_by
    GROUP BY package_id
) mc ON mc.package_id = p.package_id
WHERE EXISTS (
    SELECT 1
    FROM maintained_by mb
    JOIN maintainer m ON m.maintainer_id = mb.maintainer_id
    WHERE mb.package_id = p.package_id
    AND m.username LIKE '%lib%'
)
ORDER BY p.name;

-- #3 Query untuk mendapatkan data package yang belum pernah dimiliki oleh maintainer manapun
SELECT p.package_id, p.name
FROM package p
WHERE NOT EXISTS (
    SELECT 1
    FROM maintained_by mb 
    WHERE mb.package_id = p.package_id
)
ORDER BY p.name;

-- #4 Query untuk mendapatkan data release_file beserta hash SHA256 dari package tertentu (pencarian nama case-insensitive)
SELECT
    rf.release_file_id,
    rf.filename,
    rf.path,
    rf.size,
    rf.upload_time,
    r.release_id,
    r.version,
    fh.digest
FROM package p
JOIN release r ON r.package_id = p.package_id
JOIN release_file rf ON rf.release_id = r.release_id
JOIN file_hash fh ON fh.release_file_id = rf.release_file_id
WHERE p.name = 'numpy' COLLATE case_insensitive
AND fh.algorithm = 'SHA256';

-- #5 Query untuk mendapatkan data ringkasan jumlah release per organisasi, diurutkan dari yang terbanyak
SELECT
    o.organization_id,
    o.display_name,
    COUNT(r.release_id) AS total_release
FROM organization o
JOIN package p ON p.org_id = o.organization_id
LEFT JOIN release r ON r.package_id = p.package_id
GROUP BY o.organization_id, o.display_name
ORDER BY total_release DESC;
