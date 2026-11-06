// Package sqldb provee la lógica para crear y poblar una base de datos SQLite
// con schema tipado a partir de los CSVs del día.
// Es el equivalente interno del binario independiente csv2sqlite.
package sqldb

import (
	"fmt"
	"strings"
)

// colType clasifica cómo convertir un valor CSV al insertar en SQLite.
type colType int

const (
	colText    colType = iota
	colInteger         // int64
	colReal            // float64
)

func (t colType) sqlite() string {
	switch t {
	case colInteger:
		return "INTEGER"
	case colReal:
		return "REAL"
	default:
		return "TEXT"
	}
}

// colDef define nombre y tipo de una columna del schema.
type colDef struct {
	name string
	typ  colType
}

// fkRef declara una FK inline (column → table.column).
type fkRef struct {
	col, refTable, refCol string
}

// tableSchema describe las columnas tipadas de una tabla y su DDL.
type tableSchema struct {
	sqliteName string             // nombre de la tabla en SQLite
	csvName    string             // prefijo del archivo CSV (ej. "Kostst")
	cols       []colDef           // todas las columnas del CSV, en orden
	pk         []string           // columnas de la PK (1 o más)
	notNull    []string           // columnas con NOT NULL
	fks        []fkRef            // FKs inline
	optional   bool               // si true, se omite sin error cuando el CSV no existe
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

// indexes corre después de la carga; no depende del DDL.
const indexes = `
CREATE INDEX IF NOT EXISTS idx_art_nummer  ON ARTIKEL(ART_NUMMER);
CREATE INDEX IF NOT EXISTS idx_inv_kst     ON INVENTUR(KST_ID);
CREATE INDEX IF NOT EXISTS idx_inv_status  ON INVENTUR(INV_STATUS);
CREATE INDEX IF NOT EXISTS idx_invp_art    ON INVPOSART(ART_ID);
CREATE INDEX IF NOT EXISTS idx_invp_inv    ON INVPOSART(INV_ID);
CREATE INDEX IF NOT EXISTS idx_dt_date     ON DAILYTOTALS(DAY_DATE);
`

// buildDDL compone el DDL completo desde los schemas. Las columnas que no
// figuran en la lista del schema serían silenciosamente descartadas por el
// loader, así que esta función debe producir una tabla con TODAS las columnas
// del CSV (las nuevas se tipean como TEXT por defecto, lo que no rompe nada
// porque SQLite es de tipado dinámico).
func buildDDL() string {
	var b strings.Builder
	b.WriteString("PRAGMA foreign_keys = OFF;\n\n")
	for _, ts := range allSchemas() {
		b.WriteString(buildCreateTable(ts))
		b.WriteString("\n")
	}
	return b.String()
}

func buildCreateTable(ts *tableSchema) string {
	pkSet := make(map[string]struct{}, len(ts.pk))
	for _, c := range ts.pk {
		pkSet[c] = struct{}{}
	}
	nnSet := make(map[string]struct{}, len(ts.notNull))
	for _, c := range ts.notNull {
		nnSet[c] = struct{}{}
	}
	fkByCol := make(map[string]fkRef, len(ts.fks))
	for _, f := range ts.fks {
		fkByCol[f.col] = f
	}
	pkSingle := len(ts.pk) == 1

	var b strings.Builder
	fmt.Fprintf(&b, "CREATE TABLE IF NOT EXISTS %s (\n", ts.sqliteName)
	lines := make([]string, 0, len(ts.cols)+1)
	for _, c := range ts.cols {
		parts := []string{c.name, c.typ.sqlite()}
		if _, ok := nnSet[c.name]; ok {
			parts = append(parts, "NOT NULL")
		}
		if pkSingle && c.name == ts.pk[0] {
			parts = append(parts, "PRIMARY KEY")
		}
		if fk, ok := fkByCol[c.name]; ok {
			parts = append(parts, fmt.Sprintf("REFERENCES %s(%s)", fk.refTable, fk.refCol))
		}
		lines = append(lines, "    "+strings.Join(parts, " "))
	}
	if !pkSingle && len(ts.pk) > 0 {
		lines = append(lines, fmt.Sprintf("    PRIMARY KEY (%s)", strings.Join(ts.pk, ", ")))
	}
	b.WriteString(strings.Join(lines, ",\n"))
	b.WriteString("\n);\n")
	return b.String()
}

// Helpers para hacer las listas de columnas más cortas.
func t(name string) colDef { return colDef{name: name, typ: colText} }
func i(name string) colDef { return colDef{name: name, typ: colInteger} }
func r(name string) colDef { return colDef{name: name, typ: colReal} }

// allSchemas devuelve los schemas en el orden de carga (respeta FKs).
//
// Cada lista incluye TODAS las columnas presentes en el CSV correspondiente,
// en el mismo orden. Las columnas con tipos relevantes para el negocio
// (PKs, FKs, importes, fechas, flags) están explícitamente tipadas; el
// resto queda como TEXT por defecto (SQLite las almacena tal cual y
// generators pueden castear con db.AsInt/AsFloat al consumirlas).
func allSchemas() []*tableSchema {
	return []*tableSchema{
		kostst(),
		artikel(),
		inventur(),
		invposart(),
		hisVerbrauch(),
		hisVerbrauchpos(),
		dailytotals(),
		lieferscheinView(),
		hisLagerbew(),
		hisLagbewpos(),
		artItemCode(),
	}
}

// ── KOSTST ─────────────────────────────────────────────────────────────────
func kostst() *tableSchema {
	return &tableSchema{
		sqliteName: "KOSTST",
		csvName:    "Kostst",
		pk:         []string{"KST_ID"},
		cols: []colDef{
			i("KST_ID"), t("KST_NAME"), t("KST_INDEX"), t("KST_CODE"),
			i("KST_PARENT"), i("STE_ID"), t("KST_KONR"), r("KST_BUDGET"),
			r("KST_AUSG"), r("KST_MAX"), t("JANB_NUMMER"), i("KST_TYP"),
			i("NEW_USER"), t("NEW_ZEIT"), i("CHG_USER"), t("CHG_ZEIT"),
			t("AKTIV"), i("KST_TYPARTVK"), i("KST_TYPSPL"), i("KST_TYPLAGER"),
			t("KST_FIRMA"), t("KST_NAMEID"), t("KST_ADRESSE"), t("KST_ADD_INFO"),
			t("KST_GWV"), r("KST_GWV_A0"), r("KST_GWV_M0"), r("KST_GWV_A1"),
			r("KST_GWV_M1"), r("KST_GWV_A2"), r("KST_GWV_M2"), t("KST_VKONTO_MK"),
			t("KST_GWV_SUMLEA"), t("KST_VKBER_ID"), t("KST_VKONTO_KRED"),
			t("KST_NBA"), i("KST_NETSELL"), t("KST_NETLOGIN"), t("KST_NETSORTIMENT"),
			t("KST_NETNAME"), t("KST_NETSELLGRNR1"), t("KST_NETSELLGR1"),
			t("KST_NETSELLGRNR2"), t("KST_NETSELLGR2"), t("KST_BOCODE1"),
			t("KST_BOCODE2"), t("KST_BOCODE3"), t("KST_BOCODE4"), t("KST_BOCODE5"),
			t("KST_BOCODE6"), t("KST_BOCODE7"), t("KST_BOCODE8"),
			i("KST_CLOSEPERIOD"), i("KST_CLOSEONINV"), t("KST_DATEPERIOD"),
			i("KST_TRANSIT"), i("KST_BATCH"), i("KST_INVMETHOD"),
			i("KST_PFANDLAGER"), i("KST_INVCLOSE"), t("KST_INVTIME"),
			t("CCG_NR1"), t("CCG_NR2"), t("CCG_NR3"), t("CCG_NR4"),
			t("CCG_NR5"), t("CCG_NR6"), t("CCG_NR7"), t("CCG_NR8"),
			t("KST_WHCODE"), t("KST_EP1SRC"), t("KST_BISNR"),
			t("KST_OPENDATE"), t("KST_CLOSEDATE"), r("KST_ADDCOST"),
			i("KST_NOINVDIFF"), t("KST_EXTNAME"), i("KST_TYPPROD"),
			i("KST_NUMBER"), i("KST_CTYPE"), i("KST_LOCID"), i("KST_SPRICE"),
			i("KST_USEVSPLIST"), i("KST_INVUNITS"), i("KST_INVAUTOFILL"),
			i("KST_INVCRITICAL"), i("LOCATIONID"), i("ORGANIZATIONID"),
			i("ORGLEVELID"), t("KST_ACCOUNTGR"), t("KST_STAND"), i("KST_CPEXCLUDE"),
			t("KST_INVTMPL_ENT"), t("KST_INVTMPL_SC"), i("KST_ACTIVEIFC"),
			i("KST_USEPFAND"), i("KST_PFANDGOTO"), i("ACC_ID"), t("KST_MPLTMPL"),
			i("KST_USEFORNETORDER"), t("KST_LOCNAMEINNETS"), t("KST_NAMEINNETSELL"),
			i("KST_DEFAULTWH"), i("KST_GROUPNR"), t("KST_XREF"),
			t("KST_CITY"), t("KST_COUNTRY"), t("KST_NR2"), t("KST_PERSON"),
			t("KST_TEL"), t("KST_FAX"), t("KST_EMAIL"), i("KST_DAILYTOTAL"),
			i("KST_WAREHOUSE"), i("KST_ADVANCED_SL"), t("KST_WWW"), t("KST_STREET"),
			i("KST_PLANCAPACITY"), i("KST_ADMISSIONMEAL"), i("KST_TRANSFERMEAL"),
			i("KST_MAXADMISSIONS"), i("KST_USEFC"), i("CUR_NR"), i("KST_MYGTYPE"),
			t("KST_GUID"), i("KST_INHERITCUTOFFTIME"), t("KST_CUTOFFTIME"),
			i("KST_CUTOFFDAYS"), i("KST_DEFAULTMIL"), i("KST_REQEXTERN"),
			t("KST_ALPHCUST1"), t("KST_ALPHCUST2"), t("KST_ALPHCUST3"),
			t("KST_ALPHCUST4"), t("KST_ALPHCUST5"),
			t("KST_NUMCUST1"), t("KST_NUMCUST2"), t("KST_NUMCUST3"),
			t("KST_NUMCUST4"), t("KST_NUMCUST5"),
			t("KST_INTCUST1"), t("KST_INTCUST2"), t("KST_INTCUST3"),
			t("KST_INTCUST4"), t("KST_INTCUST5"),
			i("PSC_NR"), i("KST_LEADDAYS"), i("KND_NR"), i("KST_NR_KITCHEN"),
			i("SGP_NR"), i("KST_SORTORDER"), i("KST_PFANDDETACH"),
			i("KST_AUTOADVDAYS"), i("KST_AUTOWEEKEND"), i("KST_AUTOMODE"),
			i("KST_AUTONOMEAL"), i("KST_DIECHGMODE"), i("KST_INCLUDEICMS"),
			r("KST_DISCOUNT"), i("KST_ORDDATELOCK"), i("KST_ORDMTILOCK"),
			i("ACC_ID2"), i("KST_INVALTLAYOUT"), i("KST_INVNOPRINTPB"),
			i("KST_INVRCVOFFSET"), t("KST_ZZ"),
			i("AFR_NR1"), i("AFR_NR2"), i("AFR_NR3"),
			i("OCL_NR"), i("DIE_NR"), i("BST_NR"),
			i("KST_CLOSEFINPERIOD"), i("KST_CCORDERING"), i("KST_RSORDERING"),
			i("KST_SIMPLEHACCP"), r("KST_HACCPTEMP"), t("KST_HACCPINFO"),
			i("KST_HACCPDELTIME"), i("KST_HACCPREFRIG"), i("KST_HACCPEXPDAT"),
			i("KST_MOPASTDAYS"), i("KST_MOFUTUREDAYS"), i("PIC_NR"),
			i("KST_HACCPVEHCHK"), i("KST_ALLOWPROD"), i("KST_VTS"),
			i("KST_AUTOYIELD"), t("CCG_NR9"), t("CCG_NR10"),
			t("KST_LOCALIZATION"), i("KST_CCORDREQDIET"), i("KST_INTAKEREQ"),
			i("KST_USEENVD"), t("KST_USEENVDFROM"), i("KST_NETSDELIVERY"),
			r("KST_NETSMINORDER"), r("KST_NETSMINVALUE"), r("KST_NETSDELCHARGE"),
			i("KST_NETSDELPRO_NR"), i("SJQ_NR_CFP"), i("KST_INVMULTIHHTCNT"),
			i("MOS_NR"), i("KST_SELRESPOOLEMP"), i("KST_COUNTPURCHART"),
			i("KST_CPU"), i("KST_NR_CPU"), t("KST_TRANSFERKEY"),
			t("KST_LOCPATDATA"), t("KST_LOCPATDATAWS"),
			t("KST_ADDRESSLINE1"), t("KST_ADDRESSLINE2"), t("KST_POSTALCODE"),
			t("KST_CUSTOMERNUMBER"), i("DIE_NR_COMPANION"),
			t("KST_STATE"), t("KST_EU_IDNR"),
		},
	}
}

// ── ARTIKEL ────────────────────────────────────────────────────────────────
func artikel() *tableSchema {
	return &tableSchema{
		sqliteName: "ARTIKEL",
		csvName:    "Artikel",
		pk:         []string{"ART_ID"},
		cols: []colDef{
			i("ART_ID"), t("ART_NAME"), i("ART_NUMMER"), i("WGR_ID"),
			i("ART_TYP"), i("VPK_NR"), i("ART_LAGER"), i("ART_NR"),
			i("PGR_NR"), r("ART_VERLUST"), t("ART_NAEHRWERT"), t("ART_ABC"),
			t("ART_LTZBEST"), r("ART_OEKP"), r("ART_LTZEKP"), r("ART_VKP"),
			r("ART_MENGE"), r("ART_PROV"), i("ART_PFAND"), i("ART_FIXPREIS"),
			i("ART_NOTINV"), t("ART_CODE"), i("FBE_NR"), i("OAR_NR"),
			t("JAN_BEREICH"), t("JANA_NUMMER"), i("NEW_USER"), t("NEW_ZEIT"),
			i("CHG_USER"), t("CHG_ZEIT"), t("AKTIV"), r("ART_GWFAKTOR"),
			t("ART_INFO"), i("ART_ABVERKAUF"), t("ART_NAMEID"),
			r("ART_SOLLPREIS"), r("ART_PREIS_IVREL"), r("ART_PREIS_IVABS"),
			r("ART_MAXMENGE"), r("ART_MAXEK"), i("LF_NR"), i("VPK_NR2"),
			r("ART_MINDESTBEST"), t("ART_SBLS"), r("ART_SBLSFAKTOR"),
			i("ART_BESTVORLAUF"), i("ART_ALEVEL"), i("ART_HALTBAR"),
			t("ART_LABELS"), i("ART_FKEY"), t("ART_EAN"), t("ART_HACCP"),
			t("ART_KZGEW"), t("ART_DESC"), r("ART_DUTY"), i("ART_SHPEXP"),
			r("ART_PRICEDEV_MIN"), r("ART_PRICEDEV_MAX"),
			r("ART_QTYDEV_MIN"), r("ART_QTYDEV_MAX"),
			i("ART_EXCLUDECIL"), i("ART_BTD"), i("ART_SF"),
			t("ART_PRODUCER"), t("ART_COUNTRYORG"), t("ART_ZUSATZ"),
			i("ART_WITHDRAWL"), r("ART_VOLFACTOR"), r("ART_VOLUME"),
			i("ART_STDSTOCK"), i("ART_CREBYORDER"), i("ART_MWSTNR"),
			i("ART_OQC_METHOD"), i("ART_OQC_SUBSOH"), i("ART_OQC_ROUNDING"),
			r("ART_PRSTDLAST"), t("ART_PRSTDLASTFROM"),
			r("ART_PRSTD"), t("ART_PRSTDFROM"),
			r("ART_PRSTDNEXT"), t("ART_PRSTDNEXTFROM"),
			i("ART_PREPTYPE"), i("ART_WHOLEBATCH"), i("ART_USEVOLUME"),
			r("ART_LENGTH"), r("ART_WIDTH"), r("ART_HEIGHT"),
			t("ART_PRODNR"), i("DET_NR"), i("APPLYVC"), i("ISNOTCORPORATE"),
			i("CGP_NR"), i("ART_ISADJUSTABLE"), i("ART_ISCOLD"),
			i("ART_ISMENU"), i("ART_QUALITY"), i("ART_ISREQUEST"),
			t("ART_SHORTNAME"), i("PIC_NR"), i("ART_COUNTASREC"),
			i("ART_ISASSNEWDIET"), i("REZ_NR"), i("ART_POTISACT"),
			t("ART_GUID"), r("ART_PLANPRICE"), i("ART_SORTORDER"),
			t("ART_ALPHCUST1"), t("ART_ALPHCUST2"), t("ART_ALPHCUST3"),
			t("ART_ALPHCUST4"), t("ART_ALPHCUST5"),
			t("ART_NUMCUST1"), t("ART_NUMCUST2"), t("ART_NUMCUST3"),
			t("ART_NUMCUST4"), t("ART_NUMCUST5"),
			t("ART_INTCUST1"), t("ART_INTCUST2"), t("ART_INTCUST3"),
			t("ART_INTCUST4"), t("ART_INTCUST5"),
			t("FIT_CFOP"), t("FIT_UTILCODE"), i("CON_NR"),
			r("ART_CONTFILLBASIS"), r("ART_CONTFILLQUANT"), i("ART_ISWRITEIN"),
			i("ART_NR_BASEMENU"), i("ART_ISNOMEAL"), t("ART_INFO2"),
			t("ART_ARTFOLDER1"), t("ART_ARTFOLDER2"), i("ART_PTP_LEADDAYS"),
			i("ART_TYPE"), t("ART_PLANPRICETO"), r("ART_PLPRNEXT"),
			t("ART_PLPRNEXTFROM"), t("ART_PLPRNEXTTO"),
			i("ART_ACCEPTSHPADJ"), i("ART_PORTIONABLE"), i("ART_PREMIUM"),
			t("ART_HACCPINFO"), i("ART_AUTOYIELD"), i("ART_NUTSTATUS"),
			i("ART_NUTMETHOD"), i("PAR_NR"), r("ART_COOKEDYIELD"),
			r("ART_DIEEXCHG"), r("ART_DIEFLUID"),
			r("ART_PRICEDEV2_MIN"), r("ART_PRICEDEV2_MAX"),
			i("ART_BOOKASWASTE"), i("EFT_NR"), i("ART_SEASONABLE"),
			t("ART_SEASONS"), t("ART_PRODSPEC"), i("ART_ISDEPOSIT"),
			i("PDE_NR_VOLUME"), i("PDE_NR_WEIGHT"), r("ART_VOLUMETOWEIGHT"),
		},
	}
}

// ── INVENTUR ───────────────────────────────────────────────────────────────
func inventur() *tableSchema {
	return &tableSchema{
		sqliteName: "INVENTUR",
		csvName:    "Inventur",
		pk:         []string{"INV_ID"},
		fks:        []fkRef{{"KST_ID", "KOSTST", "KST_ID"}},
		cols: []colDef{
			i("INV_ID"), t("INV_NAME"), i("KST_ID"), t("INV_DATUM"),
			i("INV_STATUS"), i("INV_TYP"), t("INV_SELECT"), t("INV_INFO"),
			i("NEW_USER"), t("NEW_ZEIT"), i("CHG_USER"), t("CHG_ZEIT"),
			t("AKTIV"), t("INV_ACTDATUM"), t("INV_NAMEID"),
			t("INV_ZEIT"), t("INV_GENERIERT"), i("INV_CLOSEMETHOD"),
			i("REPLICATED"), i("INV_PROCESSING"), i("EXP_NR"),
			i("INV_GL_EXPORT"), t("INV_LASTPOSTED"), i("INV_ACTUSER"),
			t("INV_PARENTLIST"), i("HHT_NR"), i("INV_ENHANCED"),
			i("INV_INDIVIDUAL"), i("TFL_NR"), i("INV_COUNTPURCHART"),
			t("INV_BOOKEDAT"),
		},
	}
}

// ── INVPOSART ──────────────────────────────────────────────────────────────
//
// Importante: NO declarar PK ni FK estricta a INVENTUR — el export real tiene
// ~970/1022 filas con INV_ID huérfano (ver Estructura_Tablas_Spec sección 3.9).
func invposart() *tableSchema {
	return &tableSchema{
		sqliteName: "INVPOSART",
		csvName:    "Invposart",
		notNull:    []string{"INV_ID", "ART_ID", "VPK_ID"},
		fks: []fkRef{
			{"ART_ID", "ARTIKEL", "ART_ID"},
		},
		cols: []colDef{
			i("INV_ID"), i("INP_TYP"), i("ART_ID"), i("VPK_ID"),
			i("LGP_NR"), i("INP_SOLL"), r("INP_IST"), r("INP_ESP"),
			r("INP_EKP"), i("INP_VSP"), i("INP_VKP"), t("INP_INFO"),
			i("INP_DELART"), i("INP_STATUS"), r("INP_WEK"), r("INP_UML"),
			r("INP_KST"),
			r("INP_IST0"), r("INP_IST1"), r("INP_IST2"), r("INP_IST3"),
			r("INP_IST4"), r("INP_IST5"), r("INP_IST6"), r("INP_IST7"),
			r("INP_IST8"), r("INP_IST9"),
			r("INP_EP1"), r("INP_EP2"), t("INP_CHANGED"), i("REPLICATED"),
			r("INP_IST10"), i("SPA_NR"), i("WGR_NR"),
		},
	}
}

// ── HIS_VERBRAUCH ──────────────────────────────────────────────────────────
func hisVerbrauch() *tableSchema {
	return &tableSchema{
		sqliteName: "HIS_VERBRAUCH",
		csvName:    "His_verbrauch",
		pk:         []string{"VBR_ID"},
		fks:        []fkRef{{"KST_ID", "KOSTST", "KST_ID"}},
		cols: []colDef{
			i("VBR_ID"), t("VBR_NAME"), i("VBR_STATUS"), i("VRT_ID"),
			t("VBR_DATUM"), i("KST_ID"),
			i("NEW_USER"), t("NEW_ZEIT"), i("CHG_USER"), t("CHG_ZEIT"),
			t("AKTIV"), t("VBR_NAMEID"), t("VBR_INFO"),
			t("VBR_RC"), i("REPLICATED"), t("VBR_PPS"), t("VBR_EXP1"),
			i("EXP_NR"), i("VBR_OWNER"), i("APPLYVC"), i("MPL_NR"),
			t("VBR_PARENTLIST"), t("VBR_REQUEST"), i("VBR_FROMPOS"),
			i("VBR_ASKFORKST"), i("TFL_NR"), i("VBR_LISTLOCKED"),
			i("VBR_DEPLETED"),
		},
	}
}

// ── HIS_VERBRAUCHPOS ───────────────────────────────────────────────────────
func hisVerbrauchpos() *tableSchema {
	return &tableSchema{
		sqliteName: "HIS_VERBRAUCHPOS",
		csvName:    "His_Verbrauchpos",
		pk:         []string{"VBR_ID", "VBT_POS"},
		fks: []fkRef{
			{"VBR_ID", "HIS_VERBRAUCH", "VBR_ID"},
			{"ART_NR", "ARTIKEL", "ART_ID"},
		},
		cols: []colDef{
			i("VBR_ID"), i("VBT_POS"), i("VBT_TYP"),
			i("ART_NR"), t("REZ_NR"),
			r("VBT_MENGE"), i("VBT_WES"), r("VBT_VKP"),
			i("NEW_USER"), t("NEW_ZEIT"), i("CHG_USER"), t("CHG_ZEIT"),
			t("AKTIV"), t("VBT_INFO"), t("VBT_RC"),
			i("VPK_NR"), r("VBT_MENGEGE"), i("REPLICATED"),
			r("VBT_REQ_MENGE"), t("REZ_NR2"), i("VBT_SORT"), t("REZ_NR3"),
			t("VBT_BATCH"), t("VBT_EXPIRE"), t("VBT_HACCPINFO"),
			t("VBT_MANUFACTURED"), t("ART_NR2"), t("PRO_NR"),
		},
	}
}

// ── LIEFERSCHEIN_VIEW ──────────────────────────────────────────────────────
func lieferscheinView() *tableSchema {
	return &tableSchema{
		sqliteName: "LIEFERSCHEIN_VIEW",
		csvName:    "Lieferschein-1",
		optional:   true,
		cols: []colDef{
			t("RNG_NAME"), i("RNG_COD"), t("RNG_DATUM"),
			t("LFS_NAME"), i("LFS_RTS"), i("LF_ID"), i("LFS_ID"), t("LFS_DATUM"),
			r("LFS_NETTO"), r("LFS_MWST"), r("LFS_BRUTTO"), i("LFS_STATUS"),
			t("CHG_ZEIT"), i("ART_NR"), t("LFP_ARTNR"), i("KST_ID"),
			i("VPK_ID1"), t("POS_LFS_DATUM"),
			r("LFP_MENGE"), r("LFP_EKP"), r("LFP_VKP"),
			r("LFP_RABATT"), r("LFP_MWST"), r("LFP_BRUTTO"), r("LFP_MENGEGE"),
			t("LF_SACHB"), t("LFP_HACCPINFO"), t("LFP_ABLAUFDT"),
			t("ART_NAME"), i("ART_MUMMER"), i("ART_MUSTNR"),
		},
	}
}

// ── HIS_LAGERBEW ───────────────────────────────────────────────────────────
func hisLagerbew() *tableSchema {
	return &tableSchema{
		sqliteName: "HIS_LAGERBEW",
		csvName:    "His_lagerbew",
		pk:         []string{"LBW_ID"},
		fks: []fkRef{
			{"KST_ID", "KOSTST", "KST_ID"},
			{"KST_ID1", "KOSTST", "KST_ID"},
		},
		cols: []colDef{
			i("LBW_ID"), i("LBW_STATUS"), i("KST_ID"), i("KST_ID1"),
			t("CHG_ZEIT"),
		},
	}
}

// ── HIS_LAGBEWPOS ──────────────────────────────────────────────────────────
func hisLagbewpos() *tableSchema {
	return &tableSchema{
		sqliteName: "HIS_LAGBEWPOS",
		csvName:    "His_lagbewpos",
		pk:         []string{"LBW_ID", "LBP_POS"},
		notNull:    []string{"LBW_ID"},
		fks: []fkRef{
			{"LBW_ID", "HIS_LAGERBEW", "LBW_ID"},
			{"ART_NR", "ARTIKEL", "ART_ID"},
		},
		cols: []colDef{
			i("LBW_ID"), i("LBP_POS"), i("ART_NR"), i("VPK_ID"),
			r("LBP_MENGEGE"), r("LBP_ESP"),
		},
	}
}

// ── ART_ITEM_CODE ──────────────────────────────────────────────────────────
func artItemCode() *tableSchema {
	return &tableSchema{
		sqliteName: "ART_ITEM_CODE",
		csvName:    "Art_ItemCode",
		pk:         []string{"ART_NUMMER"},
		notNull:    []string{"ART_NUMMER"},
		cols: []colDef{
			t("ART_NAME"), i("ART_NUMMER"), i("ITEM_CODE"),
		},
	}
}

// ── DAILYTOTALS ────────────────────────────────────────────────────────────
func dailytotals() *tableSchema {
	return &tableSchema{
		sqliteName: "DAILYTOTALS",
		csvName:    "Dailytotals",
		pk:         []string{"KST_ID", "ART_ID", "DAY_DATE"},
		notNull:    []string{"KST_ID", "ART_ID", "DAY_DATE"},
		fks: []fkRef{
			{"KST_ID", "KOSTST", "KST_ID"},
			{"ART_ID", "ARTIKEL", "ART_ID"},
		},
		cols: []colDef{
			i("KST_ID"), i("ART_ID"), t("DAY_DATE"),
			r("DAY_SOHBEG"), r("DAY_SOHEND"),
			r("DAY_QTYPURCH"), r("DAY_QTYTRSFIN"), r("DAY_QTYTRSFOUT"),
			r("DAY_QTYUSAGE"), r("DAY_QTYSOLD"),
			r("DAY_QTYPRODUSE"), r("DAY_QTYPROD"),
			r("DAY_QTYINV"), r("DAY_SOHINV"), t("DAY_INVDATE"),
			r("DAY_PRICELAST"), r("DAY_PRICEAVG"), r("DAY_PRICELPURCH"),
			r("DAY_PRICELTRSF"), i("DAY_FLAGEXP"), i("DAY_FLAGPRICE"),
			i("NEW_USER"), t("NEW_ZEIT"), i("CHG_USER"), t("CHG_ZEIT"),
			r("DAY_REVENUE"), r("DAY_PRICE"),
			r("DAY_QTYINVBEG"), r("DAY_QTYEXPENSE"),
			r("DAY_PRICESTD"), r("DAY_SUMPURCH"), r("DAY_SUMTRSF"),
			r("DAY_QTYYIELDIN"), r("DAY_SUMYIELD"), r("DAY_USAGEFACTOR"),
			r("DAY_SUMVATPURCH"), r("DAY_SUMVATTRSF"), r("DAY_VATAVG"),
			i("DAY_FLAGPRICEEXP"), r("DAY_SUMTRSFOUT"),
		},
	}
}
