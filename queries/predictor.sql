-- name: ListCards :many
SELECT
    c.id,
    c.game,
    c.set_code,
    c.set_name,
    c.card_name,
    c.card_number,
    c.created_at,
    COUNT(cert.id)                                          AS cert_count,
    COUNT(cert.id) FILTER (WHERE cert.grade_received = 10)  AS psa10_count,
    COUNT(cert.id) FILTER (WHERE cert.grade_received = 9)   AS psa9_count
FROM cards c
LEFT JOIN certifications cert ON cert.card_id = c.id
GROUP BY c.id
ORDER BY c.game, c.set_name, c.card_number;

-- name: GetCard :one
SELECT id, game, set_code, set_name, card_name, card_number, created_at
FROM cards
WHERE id = $1;

-- name: CreateCard :one
INSERT INTO cards (game, set_code, set_name, card_name, card_number)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (game, set_code, card_number) DO UPDATE
    SET set_name = EXCLUDED.set_name, card_name = EXCLUDED.card_name
RETURNING *;

-- name: ListCertsForCard :many
SELECT
    cert.id,
    cert.card_id,
    cert.cert_number,
    cert.grader,
    cert.grade_received,
    cert.graded_at,
    cert.notes,
    cert.created_at,
    fi.gcs_path AS front_image,
    bi.gcs_path AS back_image,
    i.centering_front_lr,
    i.centering_front_tb,
    i.centering_back_lr,
    i.centering_back_tb,
    i.surface_front,
    i.surface_back,
    i.corner_tl,
    i.corner_tr,
    i.corner_bl,
    i.corner_br,
    i.edge_top,
    i.edge_bottom,
    i.edge_left,
    i.edge_right,
    i.source AS inspection_source
FROM certifications cert
LEFT JOIN cert_images fi ON fi.cert_id = cert.id AND fi.side = 'front'
LEFT JOIN cert_images bi ON bi.cert_id = cert.id AND bi.side = 'back'
LEFT JOIN LATERAL (
    SELECT * FROM inspections
    WHERE cert_id = cert.id
    ORDER BY created_at DESC
    LIMIT 1
) i ON true
WHERE cert.card_id = $1
ORDER BY cert.created_at DESC;

-- name: GetCert :one
SELECT id, card_id, cert_number, grader, grade_received, graded_at, notes, created_at
FROM certifications
WHERE id = $1;

-- name: CreateCert :one
INSERT INTO certifications (card_id, cert_number, grader, notes)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: SetCertGrade :one
UPDATE certifications
SET grade_received = $2, graded_at = $3
WHERE id = $1
RETURNING *;

-- name: UpsertCertImage :one
INSERT INTO cert_images (cert_id, side, gcs_path)
VALUES ($1, $2, $3)
ON CONFLICT (cert_id, side) DO UPDATE SET gcs_path = EXCLUDED.gcs_path
RETURNING *;

-- name: CreateInspection :one
INSERT INTO inspections (
    cert_id,
    centering_front_lr, centering_front_tb,
    centering_back_lr,  centering_back_tb,
    surface_front, surface_back,
    corner_tl, corner_tr, corner_bl, corner_br,
    edge_top, edge_bottom, edge_left, edge_right,
    notes, source
) VALUES (
    $1,
    $2, $3,
    $4, $5,
    $6, $7,
    $8, $9, $10, $11,
    $12, $13, $14, $15,
    $16, $17
)
RETURNING *;

-- name: ListInspectionsForCert :many
SELECT * FROM inspections
WHERE cert_id = $1
ORDER BY created_at DESC;

-- name: GetCardStats :many
-- Aggregate grade distribution grouped by simplified defect profile.
SELECT
    CASE
        WHEN i.centering_front_lr <= 55 AND i.centering_front_tb <= 55 THEN 'centered'
        WHEN i.centering_front_lr <= 60 AND i.centering_front_tb <= 60 THEN 'near_centered'
        ELSE 'off_center'
    END                 AS centering_bucket,
    COALESCE(i.surface_front, 'unknown') AS surface_front,
    COALESCE(i.surface_back, 'unknown')  AS surface_back,
    cert.grade_received,
    COUNT(*)::INT       AS cnt
FROM certifications cert
JOIN inspections i ON i.cert_id = cert.id
WHERE cert.card_id = $1
  AND cert.grade_received IS NOT NULL
GROUP BY 1, 2, 3, 4
ORDER BY cert.grade_received DESC NULLS LAST, cnt DESC;
