package tlogsql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/opessa/tlog-pipeline/internal/db"
	"github.com/opessa/tlog-pipeline/internal/tlog/common"
)

// parseFiscalDate parsea fechas en los formatos usados por la DB:
//   - Oracle DD-MON-YY: "02-JUN-26"
//   - ISO datetime:     "2006-01-02 15:04:05"
func parseFiscalDate(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	parts := strings.SplitN(s, "-", 3)
	if len(parts) == 3 && len(parts[1]) == 3 {
		mon := strings.ToUpper(parts[1][:1]) + strings.ToLower(parts[1][1:])
		if t, err := time.Parse("02-Jan-06", parts[0]+"-"+mon+"-"+parts[2]); err == nil {
			return t, true
		}
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t, true
	}
	return time.Time{}, false
}

type fiscalDocHeaderData struct {
	CAINumber                string
	CAIDate                  string
	ExemptAmount             float64
	TaxAmount                float64
	VatAmount                float64
	DifferentialIVAVatAMount float64
	IvaTaxAmount             float64
	IIBBTaxAmount            float64
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
// Los ART_NR usados dependen de h.IsProduction (ver fiscalArtNRs).
func queryFiscalDocHeaderData(ctx context.Context, conn *sql.DB, h *common.HeaderCtx, rngName string) (fiscalDocHeaderData, error) {
	var d fiscalDocHeaderData

	//	artCAI, artTax, artIva, artIIBB := fiscalArtNRs(h.IsProduction)

	const caiSQL = `
		SELECT DISTINCT lpo.LFP_HACCPINFO, lpo.LFP_ABLAUFDT
		FROM LIEFERSCHEIN_VIEW l
				 INNER JOIN LIEFERPOS lpo ON l.LFS_ID = lpo.LFS_ID
		WHERE lpo.LFP_HACCPINFO is not null and lpo.LFP_ABLAUFDT is not null
			AND l.RNG_NAME = ? AND lpo.ART_NR = 2207 AND l.LFS_STATUS = 42`
	if row, err := selectOne(ctx, conn, caiSQL, rngName); err != nil {
		return d, err
	} else if row != nil {
		d.CAINumber = row["LFP_HACCPINFO"]
		if raw := row["LFP_ABLAUFDT"]; raw != "" {
			if t, ok := parseFiscalDate(raw); ok {
				d.CAIDate = h.FormatARTimestamp(t)
			}
		}
	}

	var err error
	d.ExemptAmount, err = querySum(ctx, conn, `
		SELECT sum(l.LFP_EKP) AS val
		FROM LIEFERSCHEIN_VIEW l
				 INNER JOIN LIEFERPOS lpo ON l.LFS_ID = lpo.LFS_ID
				 INNER JOIN ARTIKEL A ON A.ART_ID = lpo.ART_NR
		WHERE l.RNG_NAME = ? AND l.LFS_STATUS = 42 AND a.ART_MWSTNR = 0 and l.ART_NR not in (2204, 2205,2206, 2207)`, rngName)
	if err != nil {
		return d, err
	}

	d.TaxAmount, err = querySum(ctx, conn, `
		SELECT sum (val) FROM  (
		SELECT l.LFP_MENGE as val
		FROM LIEFERSCHEIN_VIEW l
		WHERE l.RNG_NAME = ? AND l.ART_NR = 2204 AND l.LFS_STATUS = 42)`, rngName)
	if err != nil {
		return d, err
	}

	d.VatAmount, err = querySum(ctx, conn, `
		SELECT sum (val) FROM  (
		SELECT l.LFP_MENGE as val
		FROM LIEFERSCHEIN_VIEW l
		WHERE  l.RNG_NAME = ? AND l.ART_NR = 2255 AND l.LFS_STATUS = 42)`, rngName)
	if err != nil {
		return d, err
	}

	d.DifferentialIVAVatAMount, err = querySum(ctx, conn, `
		SELECT sum (val) FROM  (
		SELECT l.LFP_MENGE as val
		FROM LIEFERSCHEIN_VIEW l
		WHERE l.RNG_NAME = ? AND l.ART_NR = 2256 AND l.LFS_STATUS = 42)`, rngName)
	if err != nil {
		return d, err
	}

	d.IvaTaxAmount, err = querySum(ctx, conn, `
		SELECT sum (val) FROM  (
		SELECT l.LFP_MENGE as val
		FROM LIEFERSCHEIN_VIEW l
		WHERE l.RNG_NAME = ? AND l.ART_NR = 2205 AND l.LFS_STATUS = 42)`, rngName)
	if err != nil {
		return d, err
	}

	d.IIBBTaxAmount, err = querySum(ctx, conn, `
		SELECT sum (val) FROM  (
		SELECT l.LFP_MENGE as val
		FROM LIEFERSCHEIN_VIEW l
		WHERE l.RNG_NAME = ? AND l.ART_NR = 2206 AND l.LFS_STATUS = 42)`, rngName)
	if err != nil {
		return d, err
	}

	return d, nil
}

func fiscalDocReceptionLines(ctx context.Context, conn *sql.DB, rngName string) ([]map[string]string, error) {
	//	if isProd {
	const linesSQL = `
SELECT distinct lfp.ART_NR, lfp.LFS_ID, lfp.ART_NR, lfp.LFP_MENGE,
                lfp.LFP_EKP, lfp.LFP_BRUTTO, lfp.VPK_ID1, art.ART_NAME,
                art.ART_NUMMER, art.ART_MWSTNR
FROM LIEFERSCHEIN_VIEW lfp
         LEFT JOIN ARTIKEL art ON art.ART_ID = lfp.ART_NR
WHERE lfp.RNG_NAME = ? and lfp.ART_NR not in (2204, 2205,2206, 2207)
ORDER BY lfp.ART_NR, lfp.LFS_ID;
`

	rows, err := queryRows(ctx, conn, linesSQL, rngName)
	if err != nil {
		return nil, fmt.Errorf("reception lineas LFS=%s: %w", rngName, err)
	}

	return rows, nil
}
