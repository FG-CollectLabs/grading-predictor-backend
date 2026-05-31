package predictor

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Cards ────────────────────────────────────────────────────────────────────

type CardRow struct {
	ID               int64     `json:"id"`
	Game             string    `json:"game"`
	SetCode          string    `json:"set_code"`
	SetName          string    `json:"set_name"`
	CardName         string    `json:"card_name"`
	CardNumber       string    `json:"card_number"`
	CreatedAt        time.Time `json:"created_at"`
	CertCount        int64     `json:"cert_count"`
	PSA10Count       int64     `json:"psa10_count"`
	PSA9Count        int64     `json:"psa9_count"`
	ImageURL         *string   `json:"image_url"`
	MarketDisplayKey *string   `json:"market_display_key"`
}

func listCards(ctx context.Context, db *pgxpool.Pool) ([]CardRow, error) {
	rows, err := db.Query(ctx, `
		SELECT
			c.id, c.game, c.set_code, c.set_name, c.card_name, c.card_number, c.created_at,
			COUNT(cert.id),
			COUNT(cert.id) FILTER (WHERE cert.grade_received = 10),
			COUNT(cert.id) FILTER (WHERE cert.grade_received = 9),
			c.image_url, c.market_display_key
		FROM cards c
		LEFT JOIN certifications cert ON cert.card_id = c.id
		GROUP BY c.id
		ORDER BY c.game, c.set_name, c.card_number`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CardRow
	for rows.Next() {
		var r CardRow
		if err := rows.Scan(&r.ID, &r.Game, &r.SetCode, &r.SetName, &r.CardName, &r.CardNumber, &r.CreatedAt,
			&r.CertCount, &r.PSA10Count, &r.PSA9Count, &r.ImageURL, &r.MarketDisplayKey); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type CardDetail struct {
	ID               int64     `json:"id"`
	Game             string    `json:"game"`
	SetCode          string    `json:"set_code"`
	SetName          string    `json:"set_name"`
	CardName         string    `json:"card_name"`
	CardNumber       string    `json:"card_number"`
	CreatedAt        time.Time `json:"created_at"`
	ImageURL         *string   `json:"image_url"`
	MarketDisplayKey *string   `json:"market_display_key"`
}

func getCard(ctx context.Context, db *pgxpool.Pool, id int64) (*CardDetail, error) {
	var c CardDetail
	err := db.QueryRow(ctx,
		`SELECT id, game, set_code, set_name, card_name, card_number, created_at, image_url, market_display_key
		 FROM cards WHERE id = $1`, id).
		Scan(&c.ID, &c.Game, &c.SetCode, &c.SetName, &c.CardName, &c.CardNumber, &c.CreatedAt,
			&c.ImageURL, &c.MarketDisplayKey)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func createCard(ctx context.Context, db *pgxpool.Pool, game, setCode, setName, cardName, cardNumber string, imageURL, marketDisplayKey *string) (*CardDetail, error) {
	var c CardDetail
	err := db.QueryRow(ctx, `
		INSERT INTO cards (game, set_code, set_name, card_name, card_number, image_url, market_display_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (game, set_code, card_number) DO UPDATE
			SET set_name = EXCLUDED.set_name, card_name = EXCLUDED.card_name,
			    image_url = COALESCE(EXCLUDED.image_url, cards.image_url),
			    market_display_key = COALESCE(EXCLUDED.market_display_key, cards.market_display_key)
		RETURNING id, game, set_code, set_name, card_name, card_number, created_at, image_url, market_display_key`,
		game, setCode, setName, cardName, cardNumber, imageURL, marketDisplayKey).
		Scan(&c.ID, &c.Game, &c.SetCode, &c.SetName, &c.CardName, &c.CardNumber, &c.CreatedAt,
			&c.ImageURL, &c.MarketDisplayKey)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func deleteCard(ctx context.Context, db *pgxpool.Pool, id int64) (bool, error) {
	tag, err := db.Exec(ctx, `DELETE FROM cards WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// ── Certs ────────────────────────────────────────────────────────────────────

type CertRow struct {
	ID                int64      `json:"id"`
	CardID            int64      `json:"card_id"`
	CertNumber        string     `json:"cert_number"`
	Grader            string     `json:"grader"`
	GradeReceived     *int16     `json:"grade_received"`
	GradedAt          *time.Time `json:"graded_at"`
	Notes             *string    `json:"notes"`
	Category          string     `json:"category"`
	Purpose           string     `json:"purpose"`
	CreatedAt         time.Time  `json:"created_at"`
	FrontImage        *string    `json:"front_image"`
	BackImage         *string    `json:"back_image"`
	CenteringFrontLR  *int16     `json:"centering_front_lr"`
	CenteringFrontTB  *int16     `json:"centering_front_tb"`
	CenteringBackLR   *int16     `json:"centering_back_lr"`
	CenteringBackTB   *int16     `json:"centering_back_tb"`
	SurfaceFront      *string    `json:"surface_front"`
	SurfaceBack       *string    `json:"surface_back"`
	CornerTL          *string    `json:"corner_tl"`
	CornerTR          *string    `json:"corner_tr"`
	CornerBL          *string    `json:"corner_bl"`
	CornerBR          *string    `json:"corner_br"`
	EdgeTop           *string    `json:"edge_top"`
	EdgeBottom        *string    `json:"edge_bottom"`
	EdgeLeft          *string    `json:"edge_left"`
	EdgeRight         *string    `json:"edge_right"`
	InspectionSource  *string    `json:"inspection_source"`
}

func listCertsForCard(ctx context.Context, db *pgxpool.Pool, cardID int64) ([]CertRow, error) {
	rows, err := db.Query(ctx, `
		SELECT
			cert.id, cert.card_id, cert.cert_number, cert.grader,
			cert.grade_received, cert.graded_at, cert.notes,
			cert.category, cert.purpose, cert.created_at,
			fi.gcs_path, bi.gcs_path,
			i.centering_front_lr, i.centering_front_tb,
			i.centering_back_lr,  i.centering_back_tb,
			i.surface_front, i.surface_back,
			i.corner_tl, i.corner_tr, i.corner_bl, i.corner_br,
			i.edge_top, i.edge_bottom, i.edge_left, i.edge_right,
			i.source
		FROM certifications cert
		LEFT JOIN cert_images fi ON fi.cert_id = cert.id AND fi.side = 'front'
		LEFT JOIN cert_images bi ON bi.cert_id = cert.id AND bi.side = 'back'
		LEFT JOIN LATERAL (
			SELECT * FROM inspections WHERE cert_id = cert.id ORDER BY created_at DESC LIMIT 1
		) i ON true
		WHERE cert.card_id = $1
		ORDER BY cert.created_at DESC`, cardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CertRow
	for rows.Next() {
		var r CertRow
		var gradedAt pgtype.Date
		if err := rows.Scan(
			&r.ID, &r.CardID, &r.CertNumber, &r.Grader,
			&r.GradeReceived, &gradedAt, &r.Notes,
			&r.Category, &r.Purpose, &r.CreatedAt,
			&r.FrontImage, &r.BackImage,
			&r.CenteringFrontLR, &r.CenteringFrontTB,
			&r.CenteringBackLR, &r.CenteringBackTB,
			&r.SurfaceFront, &r.SurfaceBack,
			&r.CornerTL, &r.CornerTR, &r.CornerBL, &r.CornerBR,
			&r.EdgeTop, &r.EdgeBottom, &r.EdgeLeft, &r.EdgeRight,
			&r.InspectionSource,
		); err != nil {
			return nil, err
		}
		if gradedAt.Valid {
			t := gradedAt.Time
			r.GradedAt = &t
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type CertDetail struct {
	ID            int64     `json:"id"`
	CardID        int64     `json:"card_id"`
	CertNumber    string    `json:"cert_number"`
	Grader        string    `json:"grader"`
	GradeReceived *int16    `json:"grade_received"`
	GradedAt      *string   `json:"graded_at"`
	Notes         *string   `json:"notes"`
	Category      string    `json:"category"`
	Purpose       string    `json:"purpose"`
	CreatedAt     time.Time `json:"created_at"`
}

func getCert(ctx context.Context, db *pgxpool.Pool, id int64) (*CertDetail, error) {
	var c CertDetail
	var gradedAt pgtype.Date
	err := db.QueryRow(ctx,
		`SELECT id, card_id, cert_number, grader, grade_received, graded_at, notes, category, purpose, created_at
		 FROM certifications WHERE id = $1`, id).
		Scan(&c.ID, &c.CardID, &c.CertNumber, &c.Grader, &c.GradeReceived, &gradedAt, &c.Notes,
			&c.Category, &c.Purpose, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	if gradedAt.Valid {
		s := gradedAt.Time.Format("2006-01-02")
		c.GradedAt = &s
	}
	return &c, nil
}

type CertDetailResponse struct {
	ID            int64      `json:"id"`
	CardID        int64      `json:"card_id"`
	CertNumber    string     `json:"cert_number"`
	Grader        string     `json:"grader"`
	GradeReceived *int16     `json:"grade_received"`
	GradedAt      *string    `json:"graded_at"`
	Notes         *string    `json:"notes"`
	Category      string     `json:"category"`
	Purpose       string     `json:"purpose"`
	CreatedAt     time.Time  `json:"created_at"`
	FrontImage    *string    `json:"front_image"`
	BackImage     *string    `json:"back_image"`
}

func getCertDetail(ctx context.Context, db *pgxpool.Pool, id int64) (*CertDetailResponse, error) {
	var c CertDetailResponse
	var gradedAt pgtype.Date
	err := db.QueryRow(ctx, `
		SELECT cert.id, cert.card_id, cert.cert_number, cert.grader, cert.grade_received, cert.graded_at,
		       cert.notes, cert.category, cert.purpose, cert.created_at,
		       fi.gcs_path, bi.gcs_path
		FROM certifications cert
		LEFT JOIN cert_images fi ON fi.cert_id = cert.id AND fi.side = 'front'
		LEFT JOIN cert_images bi ON bi.cert_id = cert.id AND bi.side = 'back'
		WHERE cert.id = $1`, id).
		Scan(&c.ID, &c.CardID, &c.CertNumber, &c.Grader, &c.GradeReceived, &gradedAt,
			&c.Notes, &c.Category, &c.Purpose, &c.CreatedAt, &c.FrontImage, &c.BackImage)
	if err != nil {
		return nil, err
	}
	if gradedAt.Valid {
		s := gradedAt.Time.Format("2006-01-02")
		c.GradedAt = &s
	}
	return &c, nil
}

func createCert(ctx context.Context, db *pgxpool.Pool, cardID int64, certNumber, grader, notes, category, purpose string) (*CertDetail, error) {
	var notesPtr *string
	if notes != "" {
		notesPtr = &notes
	}
	if category == "" {
		category = "raw"
	}
	if purpose == "" {
		purpose = "analytics"
	}
	var c CertDetail
	var gradedAt pgtype.Date
	err := db.QueryRow(ctx,
		`INSERT INTO certifications (card_id, cert_number, grader, notes, category, purpose)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, card_id, cert_number, grader, grade_received, graded_at, notes, category, purpose, created_at`,
		cardID, certNumber, grader, notesPtr, category, purpose).
		Scan(&c.ID, &c.CardID, &c.CertNumber, &c.Grader, &c.GradeReceived, &gradedAt, &c.Notes,
			&c.Category, &c.Purpose, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	if gradedAt.Valid {
		s := gradedAt.Time.Format("2006-01-02")
		c.GradedAt = &s
	}
	return &c, nil
}

func setCertGrade(ctx context.Context, db *pgxpool.Pool, id int64, grade int16, gradedAt pgtype.Date) (*CertDetail, error) {
	var c CertDetail
	var gd pgtype.Date
	err := db.QueryRow(ctx,
		`UPDATE certifications SET grade_received = $2, graded_at = $3 WHERE id = $1
		 RETURNING id, card_id, cert_number, grader, grade_received, graded_at, notes, created_at`,
		id, grade, gradedAt).
		Scan(&c.ID, &c.CardID, &c.CertNumber, &c.Grader, &c.GradeReceived, &gd, &c.Notes, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	if gd.Valid {
		s := gd.Time.Format("2006-01-02")
		c.GradedAt = &s
	}
	return &c, nil
}

// ── Images ───────────────────────────────────────────────────────────────────

type CertImageRow struct {
	ID        int64     `json:"id"`
	CertID    int64     `json:"cert_id"`
	Side      string    `json:"side"`
	GCSPath   string    `json:"gcs_path"`
	CreatedAt time.Time `json:"created_at"`
}

func upsertCertImage(ctx context.Context, db *pgxpool.Pool, certID int64, side, gcsPath string) (*CertImageRow, error) {
	var r CertImageRow
	err := db.QueryRow(ctx, `
		INSERT INTO cert_images (cert_id, side, gcs_path)
		VALUES ($1, $2, $3)
		ON CONFLICT (cert_id, side) DO UPDATE SET gcs_path = EXCLUDED.gcs_path
		RETURNING id, cert_id, side, gcs_path, created_at`,
		certID, side, gcsPath).
		Scan(&r.ID, &r.CertID, &r.Side, &r.GCSPath, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ── Inspections ──────────────────────────────────────────────────────────────

type InspectionRow struct {
	ID               int64     `json:"id"`
	CertID           int64     `json:"cert_id"`
	CenteringFrontLR *int16    `json:"centering_front_lr"`
	CenteringFrontTB *int16    `json:"centering_front_tb"`
	CenteringBackLR  *int16    `json:"centering_back_lr"`
	CenteringBackTB  *int16    `json:"centering_back_tb"`
	SurfaceFront     *string   `json:"surface_front"`
	SurfaceBack      *string   `json:"surface_back"`
	CornerTL         *string   `json:"corner_tl"`
	CornerTR         *string   `json:"corner_tr"`
	CornerBL         *string   `json:"corner_bl"`
	CornerBR         *string   `json:"corner_br"`
	EdgeTop          *string   `json:"edge_top"`
	EdgeBottom       *string   `json:"edge_bottom"`
	EdgeLeft         *string   `json:"edge_left"`
	EdgeRight        *string   `json:"edge_right"`
	Notes            *string   `json:"notes"`
	Source           string    `json:"source"`
	CreatedAt        time.Time `json:"created_at"`
}

func createInspection(ctx context.Context, db *pgxpool.Pool, certID int64, req createInspectionRequest) (*InspectionRow, error) {
	var r InspectionRow
	err := db.QueryRow(ctx, `
		INSERT INTO inspections (
			cert_id,
			centering_front_lr, centering_front_tb,
			centering_back_lr,  centering_back_tb,
			surface_front, surface_back,
			corner_tl, corner_tr, corner_bl, corner_br,
			edge_top, edge_bottom, edge_left, edge_right,
			notes, source
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		RETURNING
			id, cert_id,
			centering_front_lr, centering_front_tb,
			centering_back_lr, centering_back_tb,
			surface_front, surface_back,
			corner_tl, corner_tr, corner_bl, corner_br,
			edge_top, edge_bottom, edge_left, edge_right,
			notes, source, created_at`,
		certID,
		req.CenteringFrontLR, req.CenteringFrontTB,
		req.CenteringBackLR, req.CenteringBackTB,
		req.SurfaceFront, req.SurfaceBack,
		req.CornerTL, req.CornerTR, req.CornerBL, req.CornerBR,
		req.EdgeTop, req.EdgeBottom, req.EdgeLeft, req.EdgeRight,
		req.Notes, req.Source,
	).Scan(
		&r.ID, &r.CertID,
		&r.CenteringFrontLR, &r.CenteringFrontTB,
		&r.CenteringBackLR, &r.CenteringBackTB,
		&r.SurfaceFront, &r.SurfaceBack,
		&r.CornerTL, &r.CornerTR, &r.CornerBL, &r.CornerBR,
		&r.EdgeTop, &r.EdgeBottom, &r.EdgeLeft, &r.EdgeRight,
		&r.Notes, &r.Source, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func listInspectionsForCert(ctx context.Context, db *pgxpool.Pool, certID int64) ([]InspectionRow, error) {
	rows, err := db.Query(ctx, `
		SELECT
			id, cert_id,
			centering_front_lr, centering_front_tb,
			centering_back_lr, centering_back_tb,
			surface_front, surface_back,
			corner_tl, corner_tr, corner_bl, corner_br,
			edge_top, edge_bottom, edge_left, edge_right,
			notes, source, created_at
		FROM inspections WHERE cert_id = $1 ORDER BY created_at DESC`, certID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []InspectionRow
	for rows.Next() {
		var r InspectionRow
		if err := rows.Scan(
			&r.ID, &r.CertID,
			&r.CenteringFrontLR, &r.CenteringFrontTB,
			&r.CenteringBackLR, &r.CenteringBackTB,
			&r.SurfaceFront, &r.SurfaceBack,
			&r.CornerTL, &r.CornerTR, &r.CornerBL, &r.CornerBR,
			&r.EdgeTop, &r.EdgeBottom, &r.EdgeLeft, &r.EdgeRight,
			&r.Notes, &r.Source, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ── Stats ────────────────────────────────────────────────────────────────────

type StatRow struct {
	CenteringBucket string `json:"centering_bucket"`
	SurfaceFront    string `json:"surface_front"`
	SurfaceBack     string `json:"surface_back"`
	GradeReceived   *int16 `json:"grade_received"`
	Count           int    `json:"count"`
}

func getCardStats(ctx context.Context, db *pgxpool.Pool, cardID int64) ([]StatRow, error) {
	rows, err := db.Query(ctx, `
		SELECT
			CASE
				WHEN i.centering_front_lr <= 55 AND i.centering_front_tb <= 55 THEN 'centered'
				WHEN i.centering_front_lr <= 60 AND i.centering_front_tb <= 60 THEN 'near_centered'
				ELSE 'off_center'
			END,
			COALESCE(i.surface_front, 'unknown'),
			COALESCE(i.surface_back, 'unknown'),
			cert.grade_received,
			COUNT(*)::INT
		FROM certifications cert
		JOIN inspections i ON i.cert_id = cert.id
		WHERE cert.card_id = $1
		  AND cert.grade_received IS NOT NULL
		GROUP BY 1, 2, 3, 4
		ORDER BY cert.grade_received DESC NULLS LAST, COUNT(*) DESC`, cardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []StatRow
	for rows.Next() {
		var r StatRow
		if err := rows.Scan(&r.CenteringBucket, &r.SurfaceFront, &r.SurfaceBack, &r.GradeReceived, &r.Count); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
