package tlogsql

import (
	"context"
	"database/sql"
	"time"

	"github.com/opessa/tlog-pipeline/internal/db"
	"github.com/opessa/tlog-pipeline/internal/tlog/common"
)

type fiscalDocHeaderData struct {
	CAINumber     string
	CAIDate       string
	ExemptAmount  float64
	TaxAmount     float64
	VatAmount     float64
	IvaTaxAmount  float64
	IIBBTaxAmount float64
}

// querySum ejecuta una query que devuelve una sola fila con un solo valor numérico.
// Si no hay filas o el valor es NULL devuelve 0.
func querySum(ctx context.Context, conn *sql.DB, query string, args ...any) (float64, error) {
	row, err := selectOne(ctx, conn, query, args...)
	if err != nil {
		return 0, err
	}
	if row == nil {
		return 0, nil
	}
	for _, v := range row {
		val, _ := db.AsFloat(v)
		return val, nil
	}
	return 0, nil
}

// queryFiscalDocHeaderData ejecuta las queries auxiliares para cada LFS_ID y
// devuelve los valores de cabecera del fiscal doc (CAI, montos por tipo de IVA).
func queryFiscalDocHeaderData(ctx context.Context, conn *sql.DB, h *common.HeaderCtx, lfsID string) (fiscalDocHeaderData, error) {
	var d fiscalDocHeaderData

	const caiSQL = `
		SELECT lpo.LFP_HACCPINFO, lpo.LFP_ABLAUFDT
		FROM LIEFERSCHEIN l
			INNER JOIN LIEFERPOS lpo ON l.LFS_ID = lpo.LFS_ID
		WHERE l.LFS_ID = ? AND lpo.ART_NR = 1120 AND l.LFS_STATUS = 42`
	if row, err := selectOne(ctx, conn, caiSQL, lfsID); err != nil {
		return d, err
	} else if row != nil {
		d.CAINumber = row["LFP_HACCPINFO"]
		if raw := row["LFP_ABLAUFDT"]; raw != "" {
			if t, parseErr := time.Parse("2006-01-02 15:04:05", raw); parseErr == nil {
				d.CAIDate = h.FormatARTimestamp(t)
			}
		}
	}

	var err error
	d.ExemptAmount, err = querySum(ctx, conn, `
		SELECT sum(lpo.LFP_EKP) AS val
		FROM LIEFERSCHEIN l
			INNER JOIN LIEFERPOS lpo ON l.LFS_ID = lpo.LFS_ID
			INNER JOIN ARTIKEL A ON A.ART_ID = lpo.ART_NR
		WHERE l.LFS_ID = ? AND l.LFS_STATUS = 42 AND A.ART_MWSTNR = 0`, lfsID)
	if err != nil {
		return d, err
	}

	d.TaxAmount, err = querySum(ctx, conn, `
	SELECT sum(total) as val from (
	SELECT (lpo.LFP_EKP * lpo.LFP_MENGE) as total
		FROM LIEFERSCHEIN l
			INNER JOIN LIEFERPOS lpo ON l.LFS_ID = lpo.LFS_ID
		WHERE l.LFS_ID = ? AND lpo.ART_NR = 1100 AND l.LFS_STATUS = 42)`, lfsID)
	if err != nil {
		return d, err
	}

	d.VatAmount, err = querySum(ctx, conn, `
	SELECT sum(lpo.LFP_EKP) AS val
		FROM LIEFERSCHEIN l
			INNER JOIN LIEFERPOS lpo ON l.LFS_ID = lpo.LFS_ID
			INNER JOIN ARTIKEL A ON A.ART_ID = lpo.ART_NR
		WHERE l.LFS_ID = ? AND l.LFS_STATUS = 42 AND (A.ART_MWSTNR IS NULL OR A.ART_MWSTNR <> 0)`, lfsID)
	if err != nil {
		return d, err
	}

	d.IvaTaxAmount, err = querySum(ctx, conn, `
	SELECT sum(total) as val FROM (
		SELECT (lpo.LFP_EKP * lpo.LFP_MENGE) as total
		FROM LIEFERSCHEIN l
			INNER JOIN LIEFERPOS lpo ON l.LFS_ID = lpo.LFS_ID
		WHERE l.LFS_ID = ? AND lpo.ART_NR = 1098 AND l.LFS_STATUS = 42)`, lfsID)
	if err != nil {
		return d, err
	}

	d.IIBBTaxAmount, err = querySum(ctx, conn, `
	SELECT sum(total) as val FROM (
		SELECT (lpo.LFP_EKP * lpo.LFP_MENGE) as total
		FROM LIEFERSCHEIN l
			INNER JOIN LIEFERPOS lpo ON l.LFS_ID = lpo.LFS_ID
		WHERE l.LFS_ID = ? AND lpo.ART_NR = 1096 AND l.LFS_STATUS = 42)`, lfsID)
	if err != nil {
		return d, err
	}

	return d, nil
}
