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
		liefer(),
		warengruppe(),
		vpckeinh(),
		artikel(),
		lieferschein(),
		lieferpos(),
		rechnung(),
		rechlfs(),
		inventur(),
		invposart(),
		hisVerbrauch(),
		hisVerbrauchpos(),
		dailytotals(),
		lieferscheinView(),
		hisLagerbew(),
		hisLagbewpos(),
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

// ── LIEFER ─────────────────────────────────────────────────────────────────
func liefer() *tableSchema {
	return &tableSchema{
		sqliteName: "LIEFER",
		csvName:    "Liefer",
		pk:         []string{"LF_ID"},
		cols: []colDef{
			i("LF_ID"), t("LF_NAME"), i("STE_ID"), t("LF_BRANCHE"),
			t("LF_ADRESSE"), t("LF_PLZ"), t("LF_TEL"), i("LF_EXISTFAX"),
			t("LF_FAX"), t("LF_TELEX"), i("LF_SACHB"), t("LF_KLAP_S"),
			t("LF_VERT"), t("LF_KLAP_V"), t("LF_KDNNR"), t("LF_LCD"),
			t("LF_WCD"), t("LF_MWSTCD"), t("LF_ZBED"), t("LF_VERSAND"),
			t("LF_LBED"), t("LF_ULFD"), t("LF_UVRJ"), r("LF_PROV"),
			t("LF_KONR"), t("JANL_NUMMER"), i("NEW_USER"), t("NEW_ZEIT"),
			i("CHG_USER"), t("CHG_ZEIT"), t("AKTIV"), t("LF_NAMEID"),
			t("SCH_NR"), t("LF_EU_IDNR"), t("LF_ZWEITWAEHR"),
			t("LF_TERMIN1"), t("LF_TERMIN2"), t("LF_TERMIN3"), t("LF_TERMIN4"),
			t("LF_TERMIN5"), t("LF_TERMIN6"), t("LF_TERMIN7"),
			i("LF_GUTSCHRIFT"), i("LF_B2BORDER"), i("LF_B2BDELIVER"),
			i("LF_LOCNR"), i("CUR_NR"), t("LF_PRICEORDERDATE"), t("LF_FMTOUTPUT"),
			i("LF_FKEY"), t("LF_SHIPMENT"), t("LF_POREPORT"), i("LF_NOFIBU"),
			t("LF_CCACCOUNT"), i("LF_STORETYPE"), r("LF_STOREPERC"),
			i("SUG_NR"), i("LF_USEMENGENFAKT"), t("LF_RATING"), t("LF_TELEX2"),
			i("LF_B2BLFSUSEPQ"), i("LF_PREPORDER"), t("LF_B2BFILE"),
			t("LF_B2BPATH"), i("LF_B2BNONEWRCV"), i("LF_B2BNOEDITRCV"),
			i("LF_B2BLOCKPO"), t("LF_LFNR"), i("LF_EPROCURE"), t("LF_INVOICE"),
			t("LF_ODCVALIDTO"), i("LF_VENTOCC"), t("LF_VENDOR_GLN"),
			t("LF_BUYER_GLN"), t("LF_SUPFOLDER"), t("LF_WWW"),
			t("LF_B2BSENDSVR"), i("LF_B2BONEFILE"), t("LF_B2BCUTOFFTIME"),
			t("LF_B2BSENDTIME"), t("LF_B2BORDERFTP"), t("LF_B2BARTCATFTP"),
			t("LF_B2BRCVFTP"), t("LF_LASTSENT"), t("LF_B2BPOACKFTP"),
			t("LF_B2BPOCONFTP"), t("LF_ORDERREMINDER"), t("LF_B2BPOTEMPLATE"),
			i("LF_B2BPOCOUNTER"), i("LF_B2BUSE_OC"), r("LF_DISCOUNT"),
			i("TXT_NR"), i("LF_COD"), i("ISNOTCORPORATE"), r("LF_CORPORATEPC"),
			t("LF_B2BCATTEMPLATE"), t("LF_B2BCATPATH"), t("LF_B2BCATFILE"),
			t("LF_B2BRCVTEMPLATE"), t("LF_B2BRCVPATH"), t("LF_B2BRCVFILE"),
			t("LF_B2BORDERWS"), t("LF_B2BARTCATWS"), t("LF_B2BRCVWS"),
			t("LF_B2BPOACKWS"), t("LF_B2BPOCONFWS"), t("LF_B2BPOACKTMPL"),
			t("LF_B2BPOCNFTMPL"), t("LF_B2BPOACKFILE"), t("LF_B2BPOACKPATH"),
			t("LF_B2BPOCNFFILE"), t("LF_B2BPOCNFPATH"),
			t("LF_B2BOCPATH"), t("LF_B2BOCFILE"), t("LF_B2BOCTEMPLATE"),
			t("LF_B2BOCFTP"), t("LF_B2BOCWS"),
			t("LF_B2BRAPATH"), t("LF_B2BRAFILE"), t("LF_B2BRATEMPLATE"),
			t("LF_B2BRAFTP"), t("LF_B2BRAWS"),
			i("LF_VENDORTYPE"), r("LF_BASERED"),
			t("LF_ALPHCUST1"), t("LF_ALPHCUST2"), t("LF_ALPHCUST3"),
			t("LF_ALPHCUST4"), t("LF_ALPHCUST5"),
			t("LF_NUMCUST1"), t("LF_NUMCUST2"), t("LF_NUMCUST3"),
			t("LF_NUMCUST4"), t("LF_NUMCUST5"),
			t("LF_INTCUST1"), t("LF_INTCUST2"), t("LF_INTCUST3"),
			t("LF_INTCUST4"), t("LF_INTCUST5"),
			i("LF_USEREMADV"), i("LF_PURCHASEALL"), i("LF_B2BSPLITPO"),
			i("LF_LEADDAYS1"), i("LF_LEADDAYS2"), i("LF_LEADDAYS3"),
			i("LF_LEADDAYS4"), i("LF_LEADDAYS5"), i("LF_LEADDAYS6"),
			i("LF_LEADDAYS7"), i("LF_B2BPOSORT"), t("LF_B2BSEND_AT_OC"),
			t("LF_B2BMAILSUBJECT"), i("LF_B2BCATCOMPLETE"), i("LF_USE_CBX"),
			i("LF_B2BPOFOREIGNDB"), t("LF_B2BPONOTIFSBJT"), i("LF_B2BPOQTYIMPORT"),
			t("LF_B2BNOTEPATH"), t("LF_B2BNOTEFILE"), t("LF_B2BNOTETEMPLATE"),
			t("LF_B2BNOTEFTP"), t("LF_B2BNOTEWS"), i("LF_RCVINACTIVEPQ"),
			t("LF_B2BCATSVR"), t("LF_B2BRCVSVR"), t("LF_B2BACKSVR"),
			t("LF_B2BOCSVR"), t("LF_B2BCATNOTIF"), t("LF_B2BRCVNOTIF"),
			t("LF_B2BACKNOTIF"), t("LF_B2BOCNOTIF"),
			t("LF_B2BMASTER"), i("LF_USEB2BMASTER"), i("LF_B2BPREFERRED"),
			i("LF_DELIVDEVCHECK"), r("LF_DISCOUNT1"),
			t("LF_DSCYEARFROM"), t("LF_DSCYEARTO"),
			r("LF_PURCHVOL1"), r("LF_PURCHVOL2"), r("LF_PURCHVOL3"),
			r("LF_PURCHVOL4"), r("LF_PURCHVOL5"),
			r("LF_PURCHDISC1"), r("LF_PURCHDISC2"), r("LF_PURCHDISC3"),
			r("LF_PURCHDISC4"), r("LF_PURCHDISC5"),
			i("LF_PURCHVOLPAY"), i("LF_SETCONTRACT"),
			t("LF_ADCOSTFROM"), t("LF_ADCOSTTO"), r("LF_ADCOSTVALUE"),
			i("LF_ADCOSTPAY"), t("LF_PRODLISTFROM"), t("LF_PRODLISTTO"),
			i("TXT_NR1"), i("TXT_NR2"), i("TXT_NR3"), i("TXT_NR4"),
			i("LF_B2B_1PO_CC_DAY"), t("LF_EMAILFROMCFG"), i("LF_B2BCATCONTRACT"),
			i("LF_B2BRCVNODUPES"), i("LF_MOLONLY"),
			t("LF_POTEMPLATE"), t("LF_POTEMPLATENP"), t("LF_BIDCOVERLETTER"),
			i("KST_NR_EXTL"), i("LF_USE_EXTL"), t("LF_EXTLSVR"),
			t("LF_EXTLIMPPATH"), t("LF_EXTLIMPFILE"), t("LF_EXTLIMPTEMPLATE"),
			t("LF_EXTLIMPFTP"), t("LF_EXTLIMPWS"),
			t("LF_EXTLEXPPATH"), t("LF_EXTLEXPFILE"), t("LF_EXTLEXPTEMPLATE"),
			t("LF_EXTLEXPFTP"), t("LF_EXTLEXPWS"), i("LF_EXTLCOUNTER"),
			i("LF_B2BSPLITPOCC"), i("LF_MINORDRULE"), i("LF_MINORDOVERRIDE"),
			r("LF_MINORDTOTALQTY"), r("LF_MINORDTOTALVAL"), r("LF_MINORDTOTALWGT"),
			i("LF_B2BCATACK"), t("LF_B2BCATACKPATH"), t("LF_B2BCATACKFILE"),
			t("LF_B2BCATACKTMPL"), t("LF_B2BCATACKFTP"), t("LF_B2BCATACKWS"),
			i("LF_B2BRCVACK"), t("LF_B2BRCVACKPATH"), t("LF_B2BRCVACKFILE"),
			t("LF_B2BRCVACKTMPL"), t("LF_B2BRCVACKFTP"), t("LF_B2BRCVACKWS"),
			i("LF_B2BCATACKCNT"), i("LF_B2BRCVACKCNT"), i("LF_UPDPQDISCOUNT"),
			t("LF_KONRADD"), t("LF_IVCSUPPLIER"), i("LF_USEIVCSUPPLIER"),
			i("LF_B2BORD_SUSPEND"), i("LF_SCONTODAYS"), t("LF_B2BOC_TASKS"),
			t("LF_RECEIPTFORMAT"), t("LF_INVOICEFORMAT"), i("LF_USEPOLANGUAGE"),
			i("LF_LANGUAGEID"), t("LF_ATTFILE"), i("LF_GROUPORDER"),
			i("LF_AUTOSBI"), i("LF_OLDPRICEDAYS"), i("LF_B2BENABLELISTED"),
			i("LF_B2BQTYMATCH"), i("LF_B2BCATCHWEIGHT"), i("LF_LEADDAYS"),
			i("LF_B2BCHECKID_CAT"), i("LF_B2BCHECKID_RCV"), i("LF_RATINGBYDOC"),
			i("LF_B2BSBI"), t("LF_B2BSBIPATH"), t("LF_B2BSBIFILE"),
			t("LF_B2BSBITMPL"), t("LF_B2BSBIREPORT"), t("LF_B2BSBIBACKUPPATH"),
			t("LF_B2BSBIFTP"), t("LF_B2BSBIWS"), i("LF_B2BSBICNT"),
			i("LF_USECUSTOMERNUMBERFROMCC"), r("LF_B2B_SUBS_PRICE"),
			t("LF_BIDMAIL"), i("LF_EXT_INVOICE"), i("LF_B2B_INVOICE"),
			r("LF_NETNET_PURCHINCOME"), r("LF_NETNET_LISTINGFEE"),
			r("LF_NETNET_ADVCOSTSUB"), i("LF_PRESERVE_IMPORTED_OC"),
			t("LF_ADDRESSLINE1"), t("LF_ADDRESSLINE2"), t("LF_STREET"),
			t("LF_CITY"), t("LF_POSTALCODE"), t("LF_COUNTRY"),
			t("LF_RECEIPTFORMAT2"), t("LF_INVOICEFORMAT2"),
			t("LF_RECEIPTFORMAT3"), t("LF_INVOICEFORMAT3"),
			t("LF_RECEIPTFORMAT4"), t("LF_INVOICEFORMAT4"),
			t("LF_RECEIPTFORMAT5"), t("LF_INVOICEFORMAT5"),
			t("LF_STATE"), t("LF_EXT_RECEIPT"), t("LF_EMAILBODY"),
		},
	}
}

// ── WARENGRUPPE ────────────────────────────────────────────────────────────
func warengruppe() *tableSchema {
	return &tableSchema{
		sqliteName: "WARENGRUPPE",
		csvName:    "Warengruppe",
		pk:         []string{"WGR_ID"},
		cols: []colDef{
			i("WGR_ID"), t("WGR_NAME"), i("SPA_ID"), i("WGR_TYP"),
			i("WGR_MWSTNR"), t("WGR_KONR"), t("WGR_MWSTKO"), t("JANW_NUMMER"),
			i("CHG_USER"), t("CHG_ZEIT"), i("NEW_USER"), t("NEW_ZEIT"),
			t("AKTIV"), t("WGR_RABNR"), i("WGR_INV_SORT"), t("WGR_NAMEID"),
			t("WGR_BEDGR1"), t("WGR_BEDGR2"), i("KRT_NR"),
			t("WGR_KONR2"), t("WGR_KONRRUECK"), t("WGR_MWSTKO2"), t("WGR_MWSTKORUECK"),
			i("WGR_EP1DEF"), r("WGR_EP1PERC"), i("WGR_EP2DEF"), r("WGR_EP2PERC"),
			t("WGR_KONR3"), t("WGR_MWSTKO3"), i("WGR_FORCB"), i("WGR_NOTRANSFER"),
			i("WGR_DEPOSIT"), i("WGR_NUMBER"), i("WGR_OQC_METHOD"),
			i("WGR_OQC_BDAYS"), i("WGR_OQC_SUBSOH"), i("WGR_OQC_ADDLEAD"),
			i("WGR_OQC_ROUNDING"), r("WGR_OQC_SFACTOR"),
			t("WGR_ACCINV1"), t("WGR_ACCEXP1"), t("WGR_ACCCOS1"), t("WGR_ACCACC1"),
			t("WGR_ACCINV2"), t("WGR_ACCEXP2"), t("WGR_ACCCOS2"), t("WGR_ACCACC2"),
			t("WGR_ACCINV3"), t("WGR_ACCEXP3"), t("WGR_ACCCOS3"), t("WGR_ACCACC3"),
			t("WGR_ACCINV4"), t("WGR_ACCEXP4"), t("WGR_ACCCOS4"), t("WGR_ACCACC4"),
			t("WGR_ACCINV5"), t("WGR_ACCEXP5"), t("WGR_ACCCOS5"), t("WGR_ACCACC5"),
			t("WGR_VATINV1"), t("WGR_VATEXP1"), t("WGR_VATCOS1"), t("WGR_VATACC1"),
			t("WGR_VATINV2"), t("WGR_VATEXP2"), t("WGR_VATCOS2"), t("WGR_VATACC2"),
			t("WGR_VATINV3"), t("WGR_VATEXP3"), t("WGR_VATCOS3"), t("WGR_VATACC3"),
			t("WGR_VATINV4"), t("WGR_VATEXP4"), t("WGR_VATCOS4"), t("WGR_VATACC4"),
			t("WGR_VATINV5"), t("WGR_VATEXP5"), t("WGR_VATCOS5"), t("WGR_VATACC5"),
			i("WGR_WGRNR"), i("WGR_WGRNRCNT"), i("ART_NR"),
			i("WGR_NOREPORT"), i("WGR_ACTTHEO"), i("WGR_TENTATIVE"),
			i("WGR_FKEY"), t("WGR_KONRADD"),
			t("WGR_ACCADD1"), t("WGR_ACCADD2"), t("WGR_ACCADD3"),
			t("WGR_ACCADD4"), t("WGR_ACCADD5"), i("VPK_NR"),
		},
	}
}

// ── VPCKEINH ───────────────────────────────────────────────────────────────
func vpckeinh() *tableSchema {
	return &tableSchema{
		sqliteName: "VPCKEINH",
		csvName:    "Vpckeinh",
		pk:         []string{"VPK_ID"},
		cols: []colDef{
			i("VPK_ID"), t("VPK_NAME"), t("VPK_BSTNAME"), r("VPK_MENGE"),
			i("VPK_EINH"), r("VPK_GRMENGE"), i("VPK_GRUND"), t("VPK_INFO"),
			t("VPK_KZGRUND"), r("VPK_PFAND"), r("VPK_CHILDPFAND"), r("VPK_WERT"),
			r("VPK_MWST"), i("NEW_USER"), t("NEW_ZEIT"), i("CHG_USER"),
			t("CHG_ZEIT"), r("VPK_PROV"), t("AKTIV"), i("VPK_USEININV"),
			t("VPK_NAMEID"), i("ART_NR"), i("VPK_CONTAINERGN"), i("VPK_FKEY"),
			i("VPK_USEFORRECYIELD"),
		},
	}
}

// ── ARTIKEL ────────────────────────────────────────────────────────────────
func artikel() *tableSchema {
	return &tableSchema{
		sqliteName: "ARTIKEL",
		csvName:    "Artikel",
		pk:         []string{"ART_ID"},
		fks: []fkRef{
			{"WGR_ID", "WARENGRUPPE", "WGR_ID"},
			{"VPK_NR", "VPCKEINH", "VPK_ID"},
			{"VPK_NR2", "VPCKEINH", "VPK_ID"},
		},
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

// ── LIEFERSCHEIN ───────────────────────────────────────────────────────────
// La estructura cambió: ahora el CSV exporta una vista desnormalizada
// (cabecera + líneas de posición) sin LFS_ID como clave única.
func lieferschein() *tableSchema {
	return &tableSchema{
		sqliteName: "LIEFERSCHEIN",
		csvName:    "Lieferschein",
		fks:        []fkRef{{"LF_ID", "LIEFER", "LF_ID"}},
		cols: []colDef{
			t("RNG_NAME"), i("RNG_COD"), t("RNG_DATUM"),
			t("LFS_NAME"), i("LFS_RTS"), i("LF_ID"), t("LFS_DATUM"),
			r("LFS_NETTO"), r("LFS_MWST"), r("LFS_BRUTTO"), i("LFS_STATUS"),
			t("CHG_ZEIT"), i("ART_NR"), t("LFP_ARTNR"), i("KST_ID"),
			i("VPK_ID1"),
			r("LFP_MENGE"), r("LFP_EKP"), r("LFP_VKP"),
			r("LFP_RABATT"), r("LFP_MWST"), r("LFP_BRUTTO"),
		},
	}
}

// ── LIEFERPOS ──────────────────────────────────────────────────────────────
func lieferpos() *tableSchema {
	return &tableSchema{
		sqliteName: "LIEFERPOS",
		csvName:    "Lieferpos",
		pk:         []string{"LFS_ID", "LFP_POS"},
		notNull:    []string{"LFS_ID", "LFP_POS"},
		fks: []fkRef{
			{"KST_ID", "KOSTST", "KST_ID"},
			{"KST_ID1", "KOSTST", "KST_ID"},
			{"ART_NR", "ARTIKEL", "ART_ID"},
			{"LF_ID", "LIEFER", "LF_ID"},
			{"VPK_ID1", "VPCKEINH", "VPK_ID"},
			{"VPK_ID2", "VPCKEINH", "VPK_ID"},
		},
		cols: []colDef{
			i("LFS_ID"), i("LFP_POS"), i("LFP_LFSPOS"), i("SB_NR"),
			i("KST_ID"), i("KRT_ID"), i("KRT_NR"), i("KST_ID1"),
			i("ART_NR"), i("LF_ID"), i("VPK_ID1"),
			r("LFP_MENGE"), r("LFP_EKP"), r("LFP_VKP"),
			r("LFP_RABATT"), r("LFP_MWST"), r("LFP_BRUTTO"),
			i("LFP_MENGEGE"), i("VPK_ID2"), i("LFP_STATUS"), i("LFP_HISTORIE"),
			t("LFS_NAME"), t("LFS_DATUM"), i("BST_ID"), t("B_NAME"),
			t("BP_FREIG"), i("BST_ID1"), t("SB_NAME"), t("SB_ZE"),
			i("BP_SONDERANGEBOT"), t("BP_LTERM"),
			r("BP_MENGE"), r("BP_EKP"), i("TXT_NR"),
			r("BP_VORSCHLAG"), r("BP_VORSCHLAGDIFF"), i("BP_SONDERBEST"),
			i("FIB_AKTIV"), t("BP_INFO"), t("LFP_INFO"),
			i("NEW_USER"), t("NEW_ZEIT"), i("CHG_USER"), t("CHG_ZEIT"),
			t("AKTIV"), i("CUR_NR"), r("CUR_EXCHANGE"), r("LFP_RABATT2"),
			r("LFP_EKPSTK"), r("LFP_TEMPERATUR"), t("LFP_LIEFERZT"),
			t("LFP_KUEHLZT"), t("LFP_CHARGE"), t("LFP_ABLAUFDT"),
			i("LFP_SORT"), r("LFP_EKPSTKORIG"), t("LFP_HACCPINFO"),
			t("B_REFNO"), t("LFP_MANUFACTURED"),
			i("LFP_MENGEGE_S"), r("LFP_EKPSTK_S"), t("LFP_PCKSLIP"),
			i("LFP_QUALITY"), i("RC_NR"), r("LFP_EKPSTKFC"),
			r("LFP_FIXDISCOUNT"), r("LFP_XTRACHARGE"),
			r("LFP_EKPSTK_ACT"), r("LFP_EKPSTK_BM"), r("LFP_EKPSTK_BMACT"),
			i("TXT_ID"), t("LFP_ACCOUNT"), t("LFP_ACCVAT"), i("LFP_MWSTNR"),
			r("LFP_UNITPRICE"), r("LFP_ADJUSTMENTS"), r("LFP_BASERED"),
			t("LFP_ALPHCUST1"), t("LFP_ALPHCUST2"), t("LFP_ALPHCUST3"),
			t("LFP_NUMCUST1"), t("LFP_NUMCUST2"), t("LFP_NUMCUST3"),
			t("LFP_INTCUST1"), t("LFP_INTCUST2"), t("LFP_INTCUST3"),
			t("FIT_CFOPPOS"), t("FIT_UTILCODEPOS"), r("LFP_BASEREDVAL"),
			i("B_ID"), r("BP_RABATT"), r("BP_RABATT2"),
			r("BP_FIXDISCOUNT"), r("BP_XTRACHARGE"), r("BP_MENGEORG"),
			i("LFP_ACCOVERRULE"), t("LFP_PRODUCER"), t("LFP_COUNTRYORG"),
			i("RC_NR_PO"), r("LFP_VEHTEMP"), i("LFP_HYGCHECK"),
			t("LFP_ACTTAKEN"), r("SB_LASTPR"), r("LFP_QUALITY_TOL"),
			t("LFP_SHP_IVC"), t("LFP_SHP_IVCDATE"),
			i("BP_POS"), i("BP_NR"), i("BP_LFNR"),
			r("BP_VORSCHLAGVAT"), t("BP_DOCLINK"), i("TXT_NR2"), t("B_INFO"),
			r("BP_WEIGHT"), r("LFP_QTYRTV"), t("LFP_DOCLINK"),
			t("LFP_SHIPMENT1"), t("LFP_SHIPMENT2"), t("LFP_SHIPMENT3"),
			t("LFP_SHIPMENT4"), t("LFP_SHIPMENT5"), t("LFP_SHIPMENT6"),
			t("LFP_SHIPMENT7"), t("LFP_SHIPMENT8"), t("LFP_SHIPMENT9"),
			t("LFP_SHIPMENT10"),
			i("VPK_NR1"), i("EFD_NR"), i("LFP_POS_CONTAINED"),
			i("ART_NR_CONTAINED"), r("LFP_B2B_SUBS_PRICE"),
			i("LFP_ARTNR"), r("LFP_DEPOSIT"),
			r("LFP_ADDTAXFEE1"), r("LFP_ADDTAXFEE2"), r("LFP_ADDTAXFEE3"),
			r("BP_VORSCHLAGADJ"), r("BP_VORSCHLAGRBT"),
			r("BP_VORSCHLAGBASEREDVAL"), r("BP_VORSCHLAGRBT2"),
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
			{"VPK_ID", "VPCKEINH", "VPK_ID"},
			{"WGR_NR", "WARENGRUPPE", "WGR_ID"},
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

// ── RECHNUNG ───────────────────────────────────────────────────────────────
func rechnung() *tableSchema {
	return &tableSchema{
		sqliteName: "RECHNUNG",
		csvName:    "Rechnung",
		pk:         []string{"RNG_ID"},
		fks:        []fkRef{{"LF_ID", "LIEFER", "LF_ID"}},
		cols: []colDef{
			i("RNG_ID"), t("RNG_NAME"), i("LF_ID"), t("RNG_DATUM"),
			t("RNG_QUITTNR"), r("RNG_NETTO"), r("RNG_MWST"), r("RNG_BRUTTO"),
			i("RNG_STATUS"), i("FIB_ID"), t("RNG_INFO"),
			i("NEW_USER"), t("NEW_ZEIT"), i("CHG_USER"), t("CHG_ZEIT"),
			t("AKTIV"), t("KOS_NR"), t("KOS_VON"), t("KOS_ANZ"),
			t("RNG_NAMEID"), t("RNG_BUCHPERIODE"), t("RNG_DATVALUTA"),
			t("RNG_FIBBELEGNR"), t("RNG_ESR"),
			r("RNG_SKONTOPROZ"), r("RNG_SKONTOVAL"),
			t("KST_NR"), t("RNG_DATDELIVERY"), i("REPLICATED"),
			i("RNG_TYPE"), i("RNG_SELFBILLING"), t("RNG_BENTRYDATE"),
			r("RNG_BVAT"), r("RNG_BGROSS"), i("RNG_COD"), i("RNG_PROCESSING"),
			t("CBX_NR1"), t("CBX_NR2"), t("CBX_NR3"), t("CBX_NR4"),
			t("RNG_DOCLINK"), i("RNG_BRUTTOCALC"), i("TFL_NR"), r("CUR_EXCHANGE"),
			i("RNG_B2BSBI"), i("RNG_ADJUSTED"), i("DGROWVER"),
			r("RNG_B2B_NETVALUE"), r("RNG_B2B_VATVALUE"), r("RNG_B2B_GROSSVALUE"),
		},
	}
}

// ── RECHLFS ────────────────────────────────────────────────────────────────
func rechlfs() *tableSchema {
	return &tableSchema{
		sqliteName: "RECHLFS",
		csvName:    "Rechlfs",
		pk:         []string{"RNG_ID", "LFS_ID"},
		notNull:    []string{"RNG_ID", "LFS_ID"},
		fks: []fkRef{
			{"RNG_ID", "RECHNUNG", "RNG_ID"},
		},
		cols: []colDef{
			i("RNG_ID"), i("LFS_ID"),
		},
	}
}

// ── LIEFERSCHEIN_VIEW ──────────────────────────────────────────────────────
func lieferscheinView() *tableSchema {
	return &tableSchema{
		sqliteName: "LIEFERSCHEIN_VIEW",
		csvName:    "ypf.lieferschein-1",
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
			t("ART_NAME"), i("ART_NUMMER"), i("ART_MWSTNR"),
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
			i("DAY_QTYPURCH"), i("DAY_QTYTRSFIN"), i("DAY_QTYTRSFOUT"),
			i("DAY_QTYUSAGE"), i("DAY_QTYSOLD"),
			i("DAY_QTYPRODUSE"), i("DAY_QTYPROD"),
			i("DAY_QTYINV"), i("DAY_SOHINV"), t("DAY_INVDATE"),
			r("DAY_PRICELAST"), r("DAY_PRICEAVG"), r("DAY_PRICELPURCH"),
			r("DAY_PRICELTRSF"), i("DAY_FLAGEXP"), i("DAY_FLAGPRICE"),
			i("NEW_USER"), t("NEW_ZEIT"), i("CHG_USER"), t("CHG_ZEIT"),
			r("DAY_REVENUE"), r("DAY_PRICE"),
			i("DAY_QTYINVBEG"), i("DAY_QTYEXPENSE"),
			r("DAY_PRICESTD"), r("DAY_SUMPURCH"), r("DAY_SUMTRSF"),
			r("DAY_QTYYIELDIN"), r("DAY_SUMYIELD"), r("DAY_USAGEFACTOR"),
			r("DAY_SUMVATPURCH"), r("DAY_SUMVATTRSF"), r("DAY_VATAVG"),
			i("DAY_FLAGPRICEEXP"), r("DAY_SUMTRSFOUT"),
		},
	}
}
