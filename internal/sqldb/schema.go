// Package sqldb provee la lógica para crear y poblar una base de datos SQLite
// con schema tipado a partir de los CSVs del día.
// Es el equivalente interno del binario independiente csv2sqlite.
package sqldb

// ddl es el esquema SQLite basado en Estructura_Tablas_Spec.md (Opción B).
const ddl = `
PRAGMA foreign_keys = OFF;

CREATE TABLE IF NOT EXISTS KOSTST (
    KST_ID         INTEGER PRIMARY KEY,
    KST_NAME       TEXT, KST_INDEX TEXT, KST_CODE TEXT,
    KST_PARENT     INTEGER, STE_ID INTEGER, KST_TYP INTEGER,
    KST_TYPLAGER   INTEGER, KST_NAMEID TEXT, KST_LOCID INTEGER,
    LOCATIONID     INTEGER, ORGANIZATIONID INTEGER, KST_GUID TEXT,
    NEW_USER INTEGER, NEW_ZEIT TEXT, CHG_USER INTEGER, CHG_ZEIT TEXT
);
CREATE TABLE IF NOT EXISTS LIEFER (
    LF_ID INTEGER PRIMARY KEY,
    LF_NAME TEXT, LF_NAMEID TEXT, STE_ID INTEGER, LF_ADRESSE TEXT,
    LF_SACHB INTEGER, LF_FKEY INTEGER, LF_GUTSCHRIFT INTEGER,
    LF_B2BORDER INTEGER, LF_B2BDELIVER INTEGER,
    NEW_USER INTEGER, NEW_ZEIT TEXT, CHG_USER INTEGER, CHG_ZEIT TEXT
);
CREATE TABLE IF NOT EXISTS WARENGRUPPE (
    WGR_ID INTEGER PRIMARY KEY,
    WGR_NAME TEXT, WGR_NAMEID TEXT, SPA_ID INTEGER, WGR_TYP INTEGER,
    WGR_MWSTNR INTEGER, WGR_INV_SORT INTEGER, WGR_FKEY INTEGER,
    WGR_OQC_BDAYS INTEGER,
    NEW_USER INTEGER, NEW_ZEIT TEXT, CHG_USER INTEGER, CHG_ZEIT TEXT
);
CREATE TABLE IF NOT EXISTS VPCKEINH (
    VPK_ID INTEGER PRIMARY KEY,
    VPK_NAME TEXT, VPK_BSTNAME TEXT, VPK_NAMEID TEXT,
    VPK_MENGE REAL, VPK_EINH INTEGER, VPK_GRMENGE REAL,
    VPK_GRUND INTEGER, VPK_KZGRUND TEXT, VPK_FKEY INTEGER,
    NEW_USER INTEGER, NEW_ZEIT TEXT, CHG_USER INTEGER, CHG_ZEIT TEXT
);
CREATE TABLE IF NOT EXISTS ARTIKEL (
    ART_ID INTEGER PRIMARY KEY,
    ART_NAME TEXT, ART_NAMEID TEXT, ART_NUMMER INTEGER,
    WGR_ID INTEGER REFERENCES WARENGRUPPE(WGR_ID),
    VPK_NR INTEGER REFERENCES VPCKEINH(VPK_ID),
    VPK_NR2 INTEGER REFERENCES VPCKEINH(VPK_ID),
    ART_TYP INTEGER, ART_LAGER INTEGER, ART_LTZBEST TEXT,
    ART_LTZEKP REAL, ART_VKP REAL, ART_FKEY INTEGER,
    ART_GWFAKTOR REAL, ART_PFAND INTEGER, ART_FIXPREIS INTEGER, ART_NOTINV INTEGER,
    NEW_USER INTEGER, NEW_ZEIT TEXT, CHG_USER INTEGER, CHG_ZEIT TEXT
);
CREATE TABLE IF NOT EXISTS LIEFERSCHEIN (
    LFS_ID INTEGER PRIMARY KEY,
    LFS_NAME TEXT, LFS_NAMEID TEXT,
    LF_ID INTEGER REFERENCES LIEFER(LF_ID),
    LFS_DATUM TEXT, LFS_NETTO REAL, LFS_MWST REAL, LFS_BRUTTO REAL,
    LFS_STATUS INTEGER, LFS_INFO TEXT, LFS_RTS INTEGER,
    LFS_BOOKED_BY INTEGER, LFS_BOOKED_AT TEXT, EXP_NR INTEGER, DGROWVER INTEGER,
    NEW_USER INTEGER, NEW_ZEIT TEXT, CHG_USER INTEGER, CHG_ZEIT TEXT
);
CREATE TABLE IF NOT EXISTS LIEFERPOS (
    LFS_ID INTEGER NOT NULL REFERENCES LIEFERSCHEIN(LFS_ID),
    LFP_POS INTEGER NOT NULL,
    LFP_LFSPOS INTEGER,
    KST_ID INTEGER REFERENCES KOSTST(KST_ID),
    KST_ID1 INTEGER REFERENCES KOSTST(KST_ID),
    ART_NR INTEGER REFERENCES ARTIKEL(ART_ID),
    LF_ID INTEGER REFERENCES LIEFER(LF_ID),
    VPK_ID1 INTEGER REFERENCES VPCKEINH(VPK_ID),
    VPK_ID2 INTEGER REFERENCES VPCKEINH(VPK_ID),
    LFP_MENGE INTEGER, LFP_MENGEGE INTEGER,
    LFP_EKP REAL, LFP_VKP REAL, LFP_RABATT REAL, LFP_MWST REAL, LFP_BRUTTO REAL,
    LFP_STATUS INTEGER, LFP_HISTORIE INTEGER,
    LFS_NAME TEXT, LFS_DATUM TEXT,
    NEW_USER INTEGER, NEW_ZEIT TEXT, CHG_USER INTEGER, CHG_ZEIT TEXT,
    PRIMARY KEY (LFS_ID, LFP_POS)
);
CREATE TABLE IF NOT EXISTS INVENTUR (
    INV_ID INTEGER PRIMARY KEY,
    INV_NAME TEXT, INV_NAMEID TEXT,
    KST_ID INTEGER REFERENCES KOSTST(KST_ID),
    INV_DATUM TEXT, INV_STATUS INTEGER, INV_TYP INTEGER,
    INV_SELECT TEXT, INV_INFO TEXT,
    INV_ACTDATUM TEXT, INV_ZEIT TEXT, INV_GENERIERT TEXT, INV_BOOKEDAT TEXT,
    INV_CLOSEMETHOD INTEGER, INV_PROCESSING INTEGER, INV_GL_EXPORT INTEGER, INV_ENHANCED INTEGER,
    NEW_USER INTEGER, NEW_ZEIT TEXT, CHG_USER INTEGER, CHG_ZEIT TEXT
);
CREATE TABLE IF NOT EXISTS INVPOSART (
    INV_ID INTEGER NOT NULL,
    ART_ID INTEGER NOT NULL REFERENCES ARTIKEL(ART_ID),
    VPK_ID INTEGER NOT NULL REFERENCES VPCKEINH(VPK_ID),
    INP_TYP INTEGER, INP_SOLL INTEGER,
    INP_IST REAL, INP_ESP REAL, INP_EKP REAL,
    INP_VSP INTEGER, INP_VKP INTEGER, INP_DELART INTEGER, INP_STATUS INTEGER,
    SPA_NR INTEGER,
    WGR_NR INTEGER REFERENCES WARENGRUPPE(WGR_ID)
);
CREATE TABLE IF NOT EXISTS HIS_VERBRAUCH (
    VBR_ID INTEGER PRIMARY KEY,
    VBR_NAME TEXT, VBR_NAMEID TEXT, VBR_STATUS INTEGER, VRT_ID INTEGER,
    VBR_DATUM TEXT,
    KST_ID INTEGER REFERENCES KOSTST(KST_ID),
    VBR_OWNER INTEGER, EXP_NR INTEGER,
    NEW_USER INTEGER, NEW_ZEIT TEXT, CHG_USER INTEGER, CHG_ZEIT TEXT
);
CREATE TABLE IF NOT EXISTS DAILYTOTALS1 (
    KST_ID INTEGER NOT NULL REFERENCES KOSTST(KST_ID),
    ART_ID INTEGER NOT NULL REFERENCES ARTIKEL(ART_ID),
    DAY_DATE TEXT NOT NULL,
    DAY_SOHBEG REAL, DAY_SOHEND REAL,
    DAY_QTYPURCH INTEGER, DAY_QTYTRSFIN INTEGER, DAY_QTYTRSFOUT INTEGER,
    DAY_QTYUSAGE INTEGER, DAY_QTYSOLD INTEGER, DAY_QTYINV INTEGER, DAY_SOHINV INTEGER,
    DAY_INVDATE TEXT,
    DAY_PRICELAST REAL, DAY_PRICEAVG REAL, DAY_PRICELPURCH REAL,
    DAY_PRICE REAL, DAY_PRICESTD REAL, DAY_SUMPURCH REAL,
    PRIMARY KEY (KST_ID, ART_ID, DAY_DATE)
);
`

const indexes = `
CREATE INDEX IF NOT EXISTS idx_lfs_lf      ON LIEFERSCHEIN(LF_ID);
CREATE INDEX IF NOT EXISTS idx_lfs_status  ON LIEFERSCHEIN(LFS_STATUS);
CREATE INDEX IF NOT EXISTS idx_lfs_rts     ON LIEFERSCHEIN(LFS_RTS);
CREATE INDEX IF NOT EXISTS idx_lfs_datum   ON LIEFERSCHEIN(LFS_DATUM);
CREATE INDEX IF NOT EXISTS idx_lfp_artnr   ON LIEFERPOS(ART_NR);
CREATE INDEX IF NOT EXISTS idx_lfp_kst     ON LIEFERPOS(KST_ID);
CREATE INDEX IF NOT EXISTS idx_art_wgr     ON ARTIKEL(WGR_ID);
CREATE INDEX IF NOT EXISTS idx_art_nummer  ON ARTIKEL(ART_NUMMER);
CREATE INDEX IF NOT EXISTS idx_inv_kst     ON INVENTUR(KST_ID);
CREATE INDEX IF NOT EXISTS idx_inv_status  ON INVENTUR(INV_STATUS);
CREATE INDEX IF NOT EXISTS idx_invp_art    ON INVPOSART(ART_ID);
CREATE INDEX IF NOT EXISTS idx_invp_inv    ON INVPOSART(INV_ID);
CREATE INDEX IF NOT EXISTS idx_dt_date     ON DAILYTOTALS1(DAY_DATE);
`

// colType clasifica cómo convertir un valor CSV al insertar en SQLite.
type colType int

const (
	colText    colType = iota
	colInteger         // int64
	colReal            // float64
)

// colDef define nombre y tipo de una columna del schema.
type colDef struct {
	name string
	typ  colType
}

// tableSchema describe las columnas tipadas de una tabla.
type tableSchema struct {
	sqliteName string // nombre de la tabla en SQLite
	csvName    string // prefijo del archivo CSV (ej. "Kostst")
	cols       []colDef
	index      map[string]colType // construido en init
}

func (ts *tableSchema) typeOf(col string) colType {
	if ts.index == nil {
		ts.index = make(map[string]colType, len(ts.cols))
		for _, c := range ts.cols {
			ts.index[c.name] = c.typ
		}
	}
	if t, ok := ts.index[col]; ok {
		return t
	}
	return colText
}

// allSchemas devuelve los schemas en el orden de carga (respeta FKs).
func allSchemas() []*tableSchema {
	return []*tableSchema{
		{sqliteName: "KOSTST", csvName: "Kostst", cols: []colDef{
			{"KST_ID", colInteger}, {"KST_NAME", colText}, {"KST_INDEX", colText},
			{"KST_CODE", colText}, {"KST_PARENT", colInteger}, {"STE_ID", colInteger},
			{"KST_TYP", colInteger}, {"KST_TYPLAGER", colInteger}, {"KST_NAMEID", colText},
			{"KST_LOCID", colInteger}, {"LOCATIONID", colInteger}, {"ORGANIZATIONID", colInteger},
			{"KST_GUID", colText},
			{"NEW_USER", colInteger}, {"NEW_ZEIT", colText}, {"CHG_USER", colInteger}, {"CHG_ZEIT", colText},
		}},
		{sqliteName: "LIEFER", csvName: "Liefer", cols: []colDef{
			{"LF_ID", colInteger}, {"LF_NAME", colText}, {"LF_NAMEID", colText},
			{"STE_ID", colInteger}, {"LF_ADRESSE", colText}, {"LF_SACHB", colInteger},
			{"LF_FKEY", colInteger}, {"LF_GUTSCHRIFT", colInteger},
			{"LF_B2BORDER", colInteger}, {"LF_B2BDELIVER", colInteger},
			{"NEW_USER", colInteger}, {"NEW_ZEIT", colText}, {"CHG_USER", colInteger}, {"CHG_ZEIT", colText},
		}},
		{sqliteName: "WARENGRUPPE", csvName: "Warengruppe", cols: []colDef{
			{"WGR_ID", colInteger}, {"WGR_NAME", colText}, {"WGR_NAMEID", colText},
			{"SPA_ID", colInteger}, {"WGR_TYP", colInteger}, {"WGR_MWSTNR", colInteger},
			{"WGR_INV_SORT", colInteger}, {"WGR_FKEY", colInteger}, {"WGR_OQC_BDAYS", colInteger},
			{"NEW_USER", colInteger}, {"NEW_ZEIT", colText}, {"CHG_USER", colInteger}, {"CHG_ZEIT", colText},
		}},
		{sqliteName: "VPCKEINH", csvName: "Vpckeinh", cols: []colDef{
			{"VPK_ID", colInteger}, {"VPK_NAME", colText}, {"VPK_BSTNAME", colText},
			{"VPK_NAMEID", colText}, {"VPK_MENGE", colReal}, {"VPK_EINH", colInteger},
			{"VPK_GRMENGE", colReal}, {"VPK_GRUND", colInteger}, {"VPK_KZGRUND", colText},
			{"VPK_FKEY", colInteger},
			{"NEW_USER", colInteger}, {"NEW_ZEIT", colText}, {"CHG_USER", colInteger}, {"CHG_ZEIT", colText},
		}},
		{sqliteName: "ARTIKEL", csvName: "Artikel", cols: []colDef{
			{"ART_ID", colInteger}, {"ART_NAME", colText}, {"ART_NAMEID", colText},
			{"ART_NUMMER", colInteger}, {"WGR_ID", colInteger}, {"VPK_NR", colInteger},
			{"VPK_NR2", colInteger}, {"ART_TYP", colInteger}, {"ART_LAGER", colInteger},
			{"ART_LTZBEST", colText}, {"ART_LTZEKP", colReal}, {"ART_VKP", colReal},
			{"ART_FKEY", colInteger}, {"ART_GWFAKTOR", colReal},
			{"ART_PFAND", colInteger}, {"ART_FIXPREIS", colInteger}, {"ART_NOTINV", colInteger},
			{"NEW_USER", colInteger}, {"NEW_ZEIT", colText}, {"CHG_USER", colInteger}, {"CHG_ZEIT", colText},
		}},
		{sqliteName: "LIEFERSCHEIN", csvName: "Lieferschein", cols: []colDef{
			{"LFS_ID", colInteger}, {"LFS_NAME", colText}, {"LFS_NAMEID", colText},
			{"LF_ID", colInteger}, {"LFS_DATUM", colText},
			{"LFS_NETTO", colReal}, {"LFS_MWST", colReal}, {"LFS_BRUTTO", colReal},
			{"LFS_STATUS", colInteger}, {"LFS_INFO", colText}, {"LFS_RTS", colInteger},
			{"LFS_BOOKED_BY", colInteger}, {"LFS_BOOKED_AT", colText},
			{"EXP_NR", colInteger}, {"DGROWVER", colInteger},
			{"NEW_USER", colInteger}, {"NEW_ZEIT", colText}, {"CHG_USER", colInteger}, {"CHG_ZEIT", colText},
		}},
		{sqliteName: "LIEFERPOS", csvName: "Lieferpos", cols: []colDef{
			{"LFS_ID", colInteger}, {"LFP_POS", colInteger}, {"LFP_LFSPOS", colInteger},
			{"KST_ID", colInteger}, {"KST_ID1", colInteger}, {"ART_NR", colInteger},
			{"LF_ID", colInteger}, {"VPK_ID1", colInteger}, {"VPK_ID2", colInteger},
			{"LFP_MENGE", colInteger}, {"LFP_MENGEGE", colInteger},
			{"LFP_EKP", colReal}, {"LFP_VKP", colReal}, {"LFP_RABATT", colReal},
			{"LFP_MWST", colReal}, {"LFP_BRUTTO", colReal},
			{"LFP_STATUS", colInteger}, {"LFP_HISTORIE", colInteger},
			{"LFS_NAME", colText}, {"LFS_DATUM", colText},
			{"NEW_USER", colInteger}, {"NEW_ZEIT", colText}, {"CHG_USER", colInteger}, {"CHG_ZEIT", colText},
		}},
		{sqliteName: "INVENTUR", csvName: "Inventur", cols: []colDef{
			{"INV_ID", colInteger}, {"INV_NAME", colText}, {"INV_NAMEID", colText},
			{"KST_ID", colInteger}, {"INV_DATUM", colText},
			{"INV_STATUS", colInteger}, {"INV_TYP", colInteger},
			{"INV_SELECT", colText}, {"INV_INFO", colText},
			{"INV_ACTDATUM", colText}, {"INV_ZEIT", colText}, {"INV_GENERIERT", colText},
			{"INV_BOOKEDAT", colText},
			{"INV_CLOSEMETHOD", colInteger}, {"INV_PROCESSING", colInteger},
			{"INV_GL_EXPORT", colInteger}, {"INV_ENHANCED", colInteger},
			{"NEW_USER", colInteger}, {"NEW_ZEIT", colText}, {"CHG_USER", colInteger}, {"CHG_ZEIT", colText},
		}},
		{sqliteName: "INVPOSART", csvName: "Invposart", cols: []colDef{
			{"INV_ID", colInteger}, {"ART_ID", colInteger}, {"VPK_ID", colInteger},
			{"INP_TYP", colInteger}, {"INP_SOLL", colInteger},
			{"INP_IST", colReal}, {"INP_ESP", colReal}, {"INP_EKP", colReal},
			{"INP_VSP", colInteger}, {"INP_VKP", colInteger},
			{"INP_DELART", colInteger}, {"INP_STATUS", colInteger},
			{"SPA_NR", colInteger}, {"WGR_NR", colInteger},
		}},
		{sqliteName: "HIS_VERBRAUCH", csvName: "His_verbrauch", cols: []colDef{
			{"VBR_ID", colInteger}, {"VBR_NAME", colText}, {"VBR_NAMEID", colText},
			{"VBR_STATUS", colInteger}, {"VRT_ID", colInteger}, {"VBR_DATUM", colText},
			{"KST_ID", colInteger}, {"VBR_OWNER", colInteger}, {"EXP_NR", colInteger},
			{"NEW_USER", colInteger}, {"NEW_ZEIT", colText}, {"CHG_USER", colInteger}, {"CHG_ZEIT", colText},
		}},
		{sqliteName: "DAILYTOTALS1", csvName: "Dailytotals", cols: []colDef{
			{"KST_ID", colInteger}, {"ART_ID", colInteger}, {"DAY_DATE", colText},
			{"DAY_SOHBEG", colReal}, {"DAY_SOHEND", colReal},
			{"DAY_QTYPURCH", colInteger}, {"DAY_QTYTRSFIN", colInteger}, {"DAY_QTYTRSFOUT", colInteger},
			{"DAY_QTYUSAGE", colInteger}, {"DAY_QTYSOLD", colInteger},
			{"DAY_QTYINV", colInteger}, {"DAY_SOHINV", colInteger},
			{"DAY_INVDATE", colText},
			{"DAY_PRICELAST", colReal}, {"DAY_PRICEAVG", colReal}, {"DAY_PRICELPURCH", colReal},
			{"DAY_PRICE", colReal}, {"DAY_PRICESTD", colReal}, {"DAY_SUMPURCH", colReal},
		}},
	}
}
