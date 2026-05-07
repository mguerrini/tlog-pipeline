# Referencia de Tablas — Spec para código Go (CSV → SQLite)

Documento de referencia con la estructura real de cada tabla, tipos de datos, claves y relaciones, listo para usar como spec de un programa Go que lea los CSV y los cargue en SQLite.

> **Versión actualizada**: basada en los archivos `*_20260505.csv` en `/mnt/project/`.

---

## 1. Convenciones generales

- **Encoding**: UTF-8 en todos los archivos.
- **Separador**: `,` (coma) en todos los archivos.
- **Cabecera**: primera línea = nombres de columnas.
- **Nombres de archivo**: tienen el sufijo de fecha del export (ej. `Kostst_20260505.csv`). El código Go puede aceptarlos por nombre fijo o por patrón (`{TablaCapitalize}_*.csv`).
- **Strings vacíos** en CSV → `NULL` en SQLite.
- **Tipos numéricos almacenados como string**: muchos valores monetarios vienen como texto con punto decimal (ej. `"1009.7500"`). Convertirlos a `REAL` al insertar.
- **Fechas y timestamps**: vienen como texto. Formato `YYYY-MM-DD HH:MM:SS`. Almacenar como `TEXT` para preservar el formato original.
- **Identificadores**: todos los `*_ID` son enteros.

---

## 2. Mapa de tablas

| # | Archivo CSV | Tabla SQLite | Filas | Cols | Cols con datos | Rol | Clave primaria |
|---|---|---|---|---|---|---|---|
| 1 | `Kostst_20260505.csv` | `KOSTST` | 2 | 217 | 55 | Centros de costo / EESS | `KST_ID` |
| 2 | `Liefer_20260505.csv` | `LIEFER` | 54 | 299 | 76 | Maestro de proveedores | `LF_ID` |
| 3 | `Warengruppe_20260505.csv` | `WARENGRUPPE` | 627 | 93 | 24 | Grupos de mercadería | `WGR_ID` |
| 4 | `Vpckeinh_20260505.csv` | `VPCKEINH` | 73 | 25 | 20 | Unidades de medida (UoM) | `VPK_ID` |
| 5 | `Artikel_20260505.csv` | `ARTIKEL` | 437 | 160 | 47 | Maestro de artículos | `ART_ID` |
| 6 | `Lieferschein_20260505.csv` | `LIEFERSCHEIN` | 12 | 50 | 22 | Cabecera de remitos / facturas / NCs | `LFS_ID` |
| 7 | `Lieferpos_20260505.csv` | `LIEFERPOS` | 150 | 140 | 63 | Detalle de LIEFERSCHEIN | `(LFS_ID, LFP_POS)` |
| 8 | `Inventur_20260505.csv` | `INVENTUR` | 1 | 31 | 21 | Cabecera de inventarios | `INV_ID` |
| 9 | `Invposart_20260505.csv` | `INVPOSART` | 1022 | 34 | 28 | Detalle de inventarios | `(INV_ID, ART_ID, VPK_ID)` |
| 10 | `His_verbrauch_20260505.csv` | `HIS_VERBRAUCH` | 1 | 28 | 13 | Cabecera de mermas | `VBR_ID` |
| 11 | `Dailytotals_20260505.csv` | `DAILYTOTALS1` | 438 | 40 | 38 | Totales diarios por SKU/EESS | `(KST_ID, ART_ID, DAY_DATE)` |

> Los CSVs contienen muchas columnas vacías (legado de un sistema con muchos campos opcionales). Para la carga en SQLite el código Go puede:
> - **Opción A — Carga completa**: leer el header del CSV y crear la tabla con todas las columnas como `TEXT`. Es la opción más simple y robusta.
> - **Opción B — Carga selectiva**: usar el DDL definido más abajo (sólo columnas relevantes con tipos correctos).

---

## 3. Estructura por tabla

> Sólo se listan las columnas relevantes (PK, FK, columnas con datos significativos para el negocio). El resto de columnas existe en el CSV y debe cargarse, pero no es relevante para las relaciones.

### 3.1. KOSTST (2 filas)

Centros de costo / EESS (Estaciones de Servicio) / Tiendas.

| Columna | Tipo SQLite | Rol | Notas |
|---|---|---|---|
| `KST_ID` | INTEGER | **PK** | Valores observados: 15, 16 |
| `KST_NAME` | TEXT | — | Nombre completo (ej. "CC Chacras de Coria 31252") |
| `KST_INDEX` | TEXT | — | Índice jerárquico (ej. "AAAAAAAHAA") |
| `KST_CODE` | TEXT | — | Código corto (ej. "CC 31252", "CC 100") |
| `KST_PARENT` | INTEGER | FK → KOSTST.KST_ID | Auto-referencia: jerarquía de centros (ambas filas → 14) |
| `STE_ID` | INTEGER | — | Identificador de empresa |
| `KST_TYP` | INTEGER | — | Tipo de centro |
| `KST_TYPLAGER` | INTEGER | — | Tipo de almacén (5 en muestras) |
| `KST_NAMEID` | TEXT | — | Nombre normalizado en mayúsculas |
| `KST_LOCID` | INTEGER | — | ID de ubicación (31252) |
| `LOCATIONID` | INTEGER | — | ID externo (13588) |
| `ORGANIZATIONID` | INTEGER | — | ID de organización (566580) |
| `KST_GUID` | TEXT | — | Identificador global |
| `NEW_USER`, `NEW_ZEIT`, `CHG_USER`, `CHG_ZEIT` | — | — | Auditoría |

**Total de columnas en CSV**: 217 (55 con datos).

> ⚠ **Nota sobre la jerarquía**: ambas filas tienen `KST_PARENT = 14`, pero en este CSV no existe el `KST_ID = 14`. Es un padre que vive en otro nivel del catálogo. Si se activa FK estricta sobre `KST_PARENT`, fallará. Recomendación: no declarar FK auto-referencial o cargarla como `DEFERRABLE`.

---

### 3.2. LIEFER (54 filas)

Maestro de proveedores.

| Columna | Tipo SQLite | Rol | Notas |
|---|---|---|---|
| `LF_ID` | INTEGER | **PK** | Rango -999..54 |
| `LF_NAME` | TEXT | — | Razón social (max 43 chars) |
| `LF_NAMEID` | TEXT | — | Nombre normalizado |
| `STE_ID` | INTEGER | — | Empresa |
| `LF_ADRESSE` | TEXT | — | Domicilio (mostly vacío) |
| `LF_SACHB` | INTEGER | — | Identificador fiscal grande (10-12 dígitos) — **candidato para `<Supplier>` del TLOG** |
| `LF_FKEY` | INTEGER | — | Foreign key externa (ID en otro sistema) |
| `LF_GUTSCHRIFT` | INTEGER | — | Flag (boolean) |
| `LF_B2BORDER`, `LF_B2BDELIVER` | INTEGER | — | Flags B2B |
| `NEW_USER`, `NEW_ZEIT`, `CHG_USER`, `CHG_ZEIT` | — | — | Auditoría |

**Total de columnas en CSV**: 299 (76 con datos).

---

### 3.3. WARENGRUPPE (627 filas)

Grupos de mercadería (clasificación impositiva y contable de los artículos).

| Columna | Tipo SQLite | Rol | Notas |
|---|---|---|---|
| `WGR_ID` | INTEGER | **PK** | Rango 51..677 |
| `WGR_NAME` | TEXT | — | Nombre del grupo |
| `WGR_NAMEID` | TEXT | — | Nombre normalizado |
| `SPA_ID` | INTEGER | — | Sección / rubro |
| `WGR_TYP` | INTEGER | — | Tipo |
| `WGR_MWSTNR` | INTEGER | — | Alícuota IVA |
| `WGR_INV_SORT` | INTEGER | — | Orden de inventario |
| `WGR_FKEY` | INTEGER | — | Identificador externo |
| `WGR_OQC_BDAYS` | INTEGER | — | Días de respaldo |
| `NEW_USER`, `NEW_ZEIT`, `CHG_USER`, `CHG_ZEIT` | — | — | Auditoría |

**Total de columnas en CSV**: 93 (24 con datos).

---

### 3.4. VPCKEINH (73 filas)

Unidades de medida.

| Columna | Tipo SQLite | Rol | Notas |
|---|---|---|---|
| `VPK_ID` | INTEGER | **PK** | Rango 1..119 |
| `VPK_NAME` | TEXT | — | Nombre (max 9 chars, ej. "Unidad", "Kg", "Litro") |
| `VPK_BSTNAME` | TEXT | — | Nombre comercial |
| `VPK_NAMEID` | TEXT | — | Normalizado |
| `VPK_MENGE` | REAL | — | Factor de conversión (a usar en TLOG `UomUnits`) |
| `VPK_EINH` | INTEGER | — | Familia (1=peso, 2=volumen, 3=unidades, etc.) |
| `VPK_GRMENGE` | REAL | — | Cantidad base |
| `VPK_GRUND` | INTEGER | — | UoM base de la familia |
| `VPK_KZGRUND` | TEXT | — | Flag (1 char) |
| `VPK_FKEY` | INTEGER | — | Identificador externo |
| `NEW_USER`, `NEW_ZEIT`, `CHG_USER`, `CHG_ZEIT` | — | — | Auditoría |

**Total de columnas en CSV**: 25 (20 con datos).

---

### 3.5. ARTIKEL (437 filas)

Maestro de artículos.

| Columna | Tipo SQLite | Rol | Notas |
|---|---|---|---|
| `ART_ID` | INTEGER | **PK** | Rango 1..1120 |
| `ART_NAME` | TEXT | — | Descripción del artículo (max 54 chars) |
| `ART_NAMEID` | TEXT | — | Descripción normalizada |
| `ART_NUMMER` | INTEGER | — | EAN / código de barras (puede ser muy grande, hasta 14 dígitos) |
| `WGR_ID` | INTEGER | **FK → WARENGRUPPE.WGR_ID** | Grupo de mercadería |
| `VPK_NR` | INTEGER | **FK → VPCKEINH.VPK_ID** | UoM por defecto |
| `VPK_NR2` | INTEGER | **FK → VPCKEINH.VPK_ID** | UoM secundaria |
| `ART_TYP` | INTEGER | — | Tipo |
| `ART_LAGER` | INTEGER | — | Manejo de stock (3 en todos los datos = "S1") |
| `ART_LTZBEST` | TEXT (date) | — | Fecha último pedido |
| `ART_LTZEKP` | REAL | — | Último costo de compra |
| `ART_VKP` | REAL | — | Precio de venta |
| `ART_FKEY` | INTEGER | — | EAN externo (mismo dominio que ART_NUMMER) |
| `ART_GWFAKTOR` | REAL | — | Factor de ganancia |
| `ART_PFAND`, `ART_FIXPREIS`, `ART_NOTINV` | INTEGER | — | Flags |
| `NEW_USER`, `NEW_ZEIT`, `CHG_USER`, `CHG_ZEIT` | — | — | Auditoría |

**Total de columnas en CSV**: 160 (47 con datos).

---

### 3.6. LIEFERSCHEIN (12 filas)

Cabecera de remitos / facturas / notas de crédito / devoluciones.

| Columna | Tipo SQLite | Rol | Notas |
|---|---|---|---|
| `LFS_ID` | INTEGER | **PK** | Rango 4..24 |
| `LFS_NAME` | TEXT | — | Nº de comprobante (formato `A-NNNNN-NNNNNNNN`, `F-...`, `N-...`) |
| `LFS_NAMEID` | TEXT | — | Nombre normalizado |
| `LF_ID` | INTEGER | **FK → LIEFER.LF_ID** | Proveedor (todos = 17 en muestras) |
| `LFS_DATUM` | TEXT (datetime) | — | Fecha del documento |
| `LFS_NETTO` | REAL | — | Importe neto |
| `LFS_MWST` | REAL | — | IVA |
| `LFS_BRUTTO` | REAL | — | Importe bruto (puede ser negativo en NCs/devoluciones) |
| `LFS_STATUS` | INTEGER | — | Todos en `42` = imputado en muestras |
| `LFS_INFO` | TEXT | — | Texto libre |
| `LFS_RTS` | INTEGER | — | `1` = devolución a proveedor (Return To Supplier) |
| `LFS_BOOKED_BY` | INTEGER | — | Usuario que imputó |
| `LFS_BOOKED_AT` | TEXT (datetime) | — | Timestamp de imputación |
| `EXP_NR` | INTEGER | — | Marca de exportado |
| `DGROWVER` | INTEGER | — | Versión de la fila |
| `NEW_USER`, `NEW_ZEIT`, `CHG_USER`, `CHG_ZEIT` | — | — | Auditoría |

**Total de columnas en CSV**: 50 (22 con datos).

---

### 3.7. LIEFERPOS (150 filas)

Detalle de LIEFERSCHEIN.

| Columna | Tipo SQLite | Rol | Notas |
|---|---|---|---|
| `LFS_ID` | INTEGER | **PK + FK → LIEFERSCHEIN** | Cabecera |
| `LFP_POS` | INTEGER | **PK** | Nº de línea (rango 1..23) |
| `LFP_LFSPOS` | INTEGER | — | Idem (redundante) |
| `KST_ID` | INTEGER | **FK → KOSTST.KST_ID** | EESS origen (todos = 15 en muestras) |
| `KST_ID1` | INTEGER | **FK → KOSTST.KST_ID** | EESS destino (transferencias) |
| `ART_NR` | INTEGER | **FK → ARTIKEL.ART_ID** | ⚠ El nombre `ART_NR` es engañoso: apunta a `ARTIKEL.ART_ID`, NO a `ARTIKEL.ART_NUMMER` |
| `LF_ID` | INTEGER | **FK → LIEFER.LF_ID** | Denormalizado de cabecera |
| `VPK_ID1` | INTEGER | **FK → VPCKEINH.VPK_ID** | UoM principal |
| `VPK_ID2` | INTEGER | **FK → VPCKEINH.VPK_ID** | UoM secundaria |
| `LFP_MENGE` | INTEGER | — | Cantidad (puede ser negativa en NCs/devoluciones) |
| `LFP_MENGEGE` | INTEGER | — | Cantidad en UoM secundaria |
| `LFP_EKP` | REAL | — | Costo unitario |
| `LFP_VKP` | REAL | — | Precio venta |
| `LFP_RABATT` | REAL | — | Descuento |
| `LFP_MWST` | REAL | — | IVA línea |
| `LFP_BRUTTO` | REAL | — | Costo total línea |
| `LFP_STATUS` | INTEGER | — | Estado de la línea (rango 2..34) |
| `LFP_HISTORIE` | INTEGER | — | Flag |
| `LFS_NAME`, `LFS_DATUM` | TEXT | — | Denormalizado de cabecera |
| `BST_ID`, `BST_ID1` | INTEGER | — | Usuarios |
| `B_NAME`, `SB_NAME` | TEXT | — | Nombre de la orden de compra |
| `BP_FREIG`, `SB_ZE`, `BP_LTERM` | TEXT | — | Timestamps de la orden |
| `BP_MENGE`, `BP_EKP` | — | — | Cantidad y costo de la orden |
| `NEW_USER`, `NEW_ZEIT`, `CHG_USER`, `CHG_ZEIT` | — | — | Auditoría |

**Total de columnas en CSV**: 140 (63 con datos).

---

### 3.8. INVENTUR (1 fila)

Cabecera de inventarios.

| Columna | Tipo SQLite | Rol | Notas |
|---|---|---|---|
| `INV_ID` | INTEGER | **PK** | Valor observado: 6 |
| `INV_NAME` | TEXT | — | Identificador lógico (ej. "INV2605-0006") |
| `INV_NAMEID` | TEXT | — | Idem normalizado |
| `KST_ID` | INTEGER | **FK → KOSTST.KST_ID** | EESS donde se realizó (15 en muestras) |
| `INV_DATUM` | TEXT (datetime) | — | Fecha del inventario |
| `INV_STATUS` | INTEGER | — | 8 = booked |
| `INV_TYP` | INTEGER | — | 4 = End-of-day |
| `INV_SELECT` | TEXT | — | Criterio de selección (texto largo) |
| `INV_INFO` | TEXT | — | Descripción libre |
| `INV_ACTDATUM`, `INV_ZEIT`, `INV_GENERIERT` | TEXT | — | Timestamps |
| `INV_CLOSEMETHOD`, `INV_PROCESSING`, `INV_GL_EXPORT`, `INV_ENHANCED` | INTEGER | — | Flags |
| `INV_BOOKEDAT` | TEXT (datetime) | — | Timestamp de imputación |
| `NEW_USER`, `NEW_ZEIT`, `CHG_USER`, `CHG_ZEIT` | — | — | Auditoría |

**Total de columnas en CSV**: 31 (21 con datos).

---

### 3.9. INVPOSART (1022 filas)

Detalle de inventarios.

| Columna | Tipo SQLite | Rol | Notas |
|---|---|---|---|
| `INV_ID` | INTEGER | **PK + FK → INVENTUR.INV_ID** | ⚠ Sólo 52/1022 filas matchean — ver nota abajo |
| `ART_ID` | INTEGER | **PK + FK → ARTIKEL.ART_ID** | |
| `VPK_ID` | INTEGER | **PK + FK → VPCKEINH.VPK_ID** | UoM contada |
| `INP_TYP` | INTEGER | — | Tipo (4 en todas las filas) |
| `INP_SOLL` | INTEGER | — | Stock teórico |
| `INP_IST` | REAL | — | Stock real contado |
| `INP_ESP` | REAL | — | Stock esperado |
| `INP_EKP` | REAL | — | Costo unitario |
| `INP_VSP`, `INP_VKP` | INTEGER | — | Precios de venta |
| `INP_DELART`, `INP_STATUS` | INTEGER | — | Flags |
| `INP_IST0..INP_IST10` | REAL/INTEGER | — | Conteos parciales (por hand-held) |
| `SPA_NR` | INTEGER | — | Sección |
| `WGR_NR` | INTEGER | **FK → WARENGRUPPE.WGR_ID** | Grupo de mercadería |

**Total de columnas en CSV**: 34 (28 con datos).

> ⚠ **FK huérfana**: `INV_ID` tiene 970 filas con valores que no existen en `INVENTUR` (sólo `INV_ID = 6` tiene padre; los valores 1, 4, 5 no están). El código Go debe **cargar igualmente** las filas (no descartarlas) y reportar la discrepancia. Es esperable porque el export de INVENTUR fue parcial.

---

### 3.10. HIS_VERBRAUCH (1 fila)

Cabecera de mermas / consumos.

| Columna | Tipo SQLite | Rol | Notas |
|---|---|---|---|
| `VBR_ID` | INTEGER | **PK** | Valor observado: 6 |
| `VBR_NAME` | TEXT | — | Identificador lógico (ej. "VBR2605-00006") |
| `VBR_NAMEID` | TEXT | — | Idem normalizado |
| `VBR_STATUS` | INTEGER | — | 2 = booked |
| `VRT_ID` | INTEGER | — | Tipo de movimiento |
| `VBR_DATUM` | TEXT (datetime) | — | Fecha de la merma |
| `KST_ID` | INTEGER | **FK → KOSTST.KST_ID** | EESS donde ocurrió |
| `VBR_OWNER` | INTEGER | — | Usuario propietario |
| `EXP_NR` | INTEGER | — | Marca de exportado |
| `NEW_USER`, `NEW_ZEIT`, `CHG_USER`, `CHG_ZEIT` | — | — | Auditoría |

**Total de columnas en CSV**: 28 (13 con datos).

> ⚠ No tenemos la tabla detalle (`HIS_VERBRAUCHPOS` o similar). Sin ese detalle no se pueden generar las líneas de un `InventoryAdjustment` originado en una merma.

---

### 3.11. DAILYTOTALS1 (438 filas)

Totales diarios por SKU y EESS.

| Columna | Tipo SQLite | Rol | Notas |
|---|---|---|---|
| `KST_ID` | INTEGER | **PK + FK → KOSTST.KST_ID** | EESS (15 ó 16) |
| `ART_ID` | INTEGER | **PK + FK → ARTIKEL.ART_ID** | SKU |
| `DAY_DATE` | TEXT (date) | **PK** | Fecha |
| `DAY_SOHBEG` | REAL | — | Stock inicio del día |
| `DAY_SOHEND` | REAL | — | Stock fin del día |
| `DAY_QTYPURCH` | INTEGER | — | Cantidad comprada |
| `DAY_QTYTRSFIN`, `DAY_QTYTRSFOUT` | INTEGER | — | Transferencias entrantes/salientes |
| `DAY_QTYUSAGE` | INTEGER | — | Cantidad usada |
| `DAY_QTYSOLD` | INTEGER | — | Cantidad vendida |
| `DAY_QTYINV`, `DAY_SOHINV` | INTEGER | — | Cantidad e impacto del inventario |
| `DAY_INVDATE` | TEXT (date) | — | Fecha del inventario que afectó este día |
| `DAY_PRICELAST` | REAL | — | Último precio |
| `DAY_PRICEAVG` | REAL | — | Precio promedio |
| `DAY_PRICELPURCH` | REAL | — | Último precio de compra |
| `DAY_PRICE` | REAL | — | Precio del día |
| `DAY_PRICESTD` | REAL | — | Precio estándar |
| `DAY_SUMPURCH` | REAL | — | Suma de compras |

**Total de columnas en CSV**: 40 (38 con datos).

---

## 4. Resumen de relaciones (FKs)

Validadas contra los datos reales (todas cierran 100%, salvo la indicada):

```
LIEFERPOS.LFS_ID            →  LIEFERSCHEIN.LFS_ID         ✓ 150/150
LIEFERPOS.ART_NR            →  ARTIKEL.ART_ID              ✓ 150/150  (⚠ nombre engañoso)
LIEFERPOS.VPK_ID1           →  VPCKEINH.VPK_ID             ✓ 150/150
LIEFERPOS.VPK_ID2           →  VPCKEINH.VPK_ID             ✓ 150/150
LIEFERPOS.KST_ID            →  KOSTST.KST_ID               ✓ 150/150
LIEFERPOS.KST_ID1           →  KOSTST.KST_ID               ✓ 150/150
LIEFERPOS.LF_ID             →  LIEFER.LF_ID                ✓ 150/150  (denormalizado)

LIEFERSCHEIN.LF_ID          →  LIEFER.LF_ID                ✓ 12/12

ARTIKEL.WGR_ID              →  WARENGRUPPE.WGR_ID          ✓ 437/437
ARTIKEL.VPK_NR              →  VPCKEINH.VPK_ID             ✓ 437/437
ARTIKEL.VPK_NR2             →  VPCKEINH.VPK_ID             ✓ 437/437

INVENTUR.KST_ID             →  KOSTST.KST_ID               ✓ 1/1

INVPOSART.INV_ID            →  INVENTUR.INV_ID             ⚠ 52/1022 (970 huérfanas)
INVPOSART.ART_ID            →  ARTIKEL.ART_ID              ✓ 1022/1022
INVPOSART.VPK_ID            →  VPCKEINH.VPK_ID             ✓ 1022/1022
INVPOSART.WGR_NR            →  WARENGRUPPE.WGR_ID          ✓ 1022/1022

HIS_VERBRAUCH.KST_ID        →  KOSTST.KST_ID               ✓ 1/1

DAILYTOTALS1.KST_ID         →  KOSTST.KST_ID               ✓ 438/438
DAILYTOTALS1.ART_ID         →  ARTIKEL.ART_ID              ✓ 438/438

KOSTST.KST_PARENT           →  KOSTST.KST_ID               ⚠ 0/2 (padre 14 no está en este export)
```

---

## 5. Esquema SQL completo (DDL)

> Schema sugerido para SQLite con sólo las columnas relevantes y tipos correctos. Para preservar la fidelidad del export podés cargar **todas** las columnas como `TEXT` adicionalmente.

```sql
PRAGMA foreign_keys = OFF;  -- ON al final, después de cargar todo

-- =============================================================
-- KOSTST  (centros de costo / EESS)
-- =============================================================
CREATE TABLE KOSTST (
    KST_ID         INTEGER PRIMARY KEY,
    KST_NAME       TEXT,
    KST_INDEX      TEXT,
    KST_CODE       TEXT,
    KST_PARENT     INTEGER,                  -- auto-FK no estricta (padre puede no estar)
    STE_ID         INTEGER,
    KST_TYP        INTEGER,
    KST_TYPLAGER   INTEGER,
    KST_NAMEID     TEXT,
    KST_LOCID      INTEGER,
    LOCATIONID     INTEGER,
    ORGANIZATIONID INTEGER,
    KST_GUID       TEXT,
    NEW_USER       INTEGER,
    NEW_ZEIT       TEXT,
    CHG_USER       INTEGER,
    CHG_ZEIT       TEXT
);

-- =============================================================
-- LIEFER  (proveedores)
-- =============================================================
CREATE TABLE LIEFER (
    LF_ID         INTEGER PRIMARY KEY,
    LF_NAME       TEXT,
    LF_NAMEID     TEXT,
    STE_ID        INTEGER,
    LF_ADRESSE    TEXT,
    LF_SACHB      INTEGER,
    LF_FKEY       INTEGER,
    LF_GUTSCHRIFT INTEGER,
    LF_B2BORDER   INTEGER,
    LF_B2BDELIVER INTEGER,
    NEW_USER      INTEGER,
    NEW_ZEIT      TEXT,
    CHG_USER      INTEGER,
    CHG_ZEIT      TEXT
);

-- =============================================================
-- WARENGRUPPE  (grupos de mercadería)
-- =============================================================
CREATE TABLE WARENGRUPPE (
    WGR_ID         INTEGER PRIMARY KEY,
    WGR_NAME       TEXT,
    WGR_NAMEID     TEXT,
    SPA_ID         INTEGER,
    WGR_TYP        INTEGER,
    WGR_MWSTNR     INTEGER,
    WGR_INV_SORT   INTEGER,
    WGR_FKEY       INTEGER,
    WGR_OQC_BDAYS  INTEGER,
    NEW_USER       INTEGER,
    NEW_ZEIT       TEXT,
    CHG_USER       INTEGER,
    CHG_ZEIT       TEXT
);

-- =============================================================
-- VPCKEINH  (unidades de medida)
-- =============================================================
CREATE TABLE VPCKEINH (
    VPK_ID       INTEGER PRIMARY KEY,
    VPK_NAME     TEXT,
    VPK_BSTNAME  TEXT,
    VPK_NAMEID   TEXT,
    VPK_MENGE    REAL,
    VPK_EINH     INTEGER,
    VPK_GRMENGE  REAL,
    VPK_GRUND    INTEGER,
    VPK_KZGRUND  TEXT,
    VPK_FKEY     INTEGER,
    NEW_USER     INTEGER,
    NEW_ZEIT     TEXT,
    CHG_USER     INTEGER,
    CHG_ZEIT     TEXT
);

-- =============================================================
-- ARTIKEL  (artículos)
-- =============================================================
CREATE TABLE ARTIKEL (
    ART_ID        INTEGER PRIMARY KEY,
    ART_NAME      TEXT,
    ART_NAMEID    TEXT,
    ART_NUMMER    INTEGER,                   -- EAN
    WGR_ID        INTEGER REFERENCES WARENGRUPPE(WGR_ID),
    VPK_NR        INTEGER REFERENCES VPCKEINH(VPK_ID),
    VPK_NR2       INTEGER REFERENCES VPCKEINH(VPK_ID),
    ART_TYP       INTEGER,
    ART_LAGER     INTEGER,
    ART_LTZBEST   TEXT,
    ART_LTZEKP    REAL,
    ART_VKP       REAL,
    ART_FKEY      INTEGER,
    ART_GWFAKTOR  REAL,
    ART_PFAND     INTEGER,
    ART_FIXPREIS  INTEGER,
    ART_NOTINV    INTEGER,
    NEW_USER      INTEGER,
    NEW_ZEIT      TEXT,
    CHG_USER      INTEGER,
    CHG_ZEIT      TEXT
);

-- =============================================================
-- LIEFERSCHEIN  (cabecera de remitos / facturas / NCs)
-- =============================================================
CREATE TABLE LIEFERSCHEIN (
    LFS_ID         INTEGER PRIMARY KEY,
    LFS_NAME       TEXT,
    LFS_NAMEID     TEXT,
    LF_ID          INTEGER REFERENCES LIEFER(LF_ID),
    LFS_DATUM      TEXT,
    LFS_NETTO      REAL,
    LFS_MWST       REAL,
    LFS_BRUTTO     REAL,
    LFS_STATUS     INTEGER,
    LFS_INFO       TEXT,
    LFS_RTS        INTEGER,
    LFS_BOOKED_BY  INTEGER,
    LFS_BOOKED_AT  TEXT,
    EXP_NR         INTEGER,
    DGROWVER       INTEGER,
    NEW_USER       INTEGER,
    NEW_ZEIT       TEXT,
    CHG_USER       INTEGER,
    CHG_ZEIT       TEXT
);

-- =============================================================
-- LIEFERPOS  (detalle)
-- =============================================================
CREATE TABLE LIEFERPOS (
    LFS_ID       INTEGER NOT NULL REFERENCES LIEFERSCHEIN(LFS_ID),
    LFP_POS      INTEGER NOT NULL,
    LFP_LFSPOS   INTEGER,
    KST_ID       INTEGER REFERENCES KOSTST(KST_ID),
    KST_ID1      INTEGER REFERENCES KOSTST(KST_ID),
    ART_NR       INTEGER REFERENCES ARTIKEL(ART_ID),  -- ⚠ nombre engañoso
    LF_ID        INTEGER REFERENCES LIEFER(LF_ID),
    VPK_ID1      INTEGER REFERENCES VPCKEINH(VPK_ID),
    VPK_ID2      INTEGER REFERENCES VPCKEINH(VPK_ID),
    LFP_MENGE    INTEGER,
    LFP_MENGEGE  INTEGER,
    LFP_EKP      REAL,
    LFP_VKP      REAL,
    LFP_RABATT   REAL,
    LFP_MWST     REAL,
    LFP_BRUTTO   REAL,
    LFP_STATUS   INTEGER,
    LFP_HISTORIE INTEGER,
    LFS_NAME     TEXT,
    LFS_DATUM    TEXT,
    NEW_USER     INTEGER,
    NEW_ZEIT     TEXT,
    CHG_USER     INTEGER,
    CHG_ZEIT     TEXT,
    PRIMARY KEY (LFS_ID, LFP_POS)
);

-- =============================================================
-- INVENTUR  (cabecera de inventarios)
-- =============================================================
CREATE TABLE INVENTUR (
    INV_ID         INTEGER PRIMARY KEY,
    INV_NAME       TEXT,
    INV_NAMEID     TEXT,
    KST_ID         INTEGER REFERENCES KOSTST(KST_ID),
    INV_DATUM      TEXT,
    INV_STATUS     INTEGER,
    INV_TYP        INTEGER,
    INV_SELECT     TEXT,
    INV_INFO       TEXT,
    INV_ACTDATUM   TEXT,
    INV_ZEIT       TEXT,
    INV_GENERIERT  TEXT,
    INV_BOOKEDAT   TEXT,
    INV_CLOSEMETHOD INTEGER,
    INV_PROCESSING INTEGER,
    INV_GL_EXPORT  INTEGER,
    INV_ENHANCED   INTEGER,
    NEW_USER       INTEGER,
    NEW_ZEIT       TEXT,
    CHG_USER       INTEGER,
    CHG_ZEIT       TEXT
);

-- =============================================================
-- INVPOSART  (detalle de inventarios)
-- =============================================================
-- Nota: NO declarar FK estricta a INVENTUR (970/1022 filas son huérfanas en este export)
CREATE TABLE INVPOSART (
    INV_ID    INTEGER NOT NULL,
    ART_ID    INTEGER NOT NULL REFERENCES ARTIKEL(ART_ID),
    VPK_ID    INTEGER NOT NULL REFERENCES VPCKEINH(VPK_ID),
    INP_TYP   INTEGER,
    INP_SOLL  INTEGER,
    INP_IST   REAL,
    INP_ESP   REAL,
    INP_EKP   REAL,
    INP_VSP   INTEGER,
    INP_VKP   INTEGER,
    INP_DELART INTEGER,
    INP_STATUS INTEGER,
    SPA_NR    INTEGER,
    WGR_NR    INTEGER REFERENCES WARENGRUPPE(WGR_ID),
    PRIMARY KEY (INV_ID, ART_ID, VPK_ID)
);

-- =============================================================
-- HIS_VERBRAUCH  (cabecera de mermas)
-- =============================================================
CREATE TABLE HIS_VERBRAUCH (
    VBR_ID      INTEGER PRIMARY KEY,
    VBR_NAME    TEXT,
    VBR_NAMEID  TEXT,
    VBR_STATUS  INTEGER,
    VRT_ID      INTEGER,
    VBR_DATUM   TEXT,
    KST_ID      INTEGER REFERENCES KOSTST(KST_ID),
    VBR_OWNER   INTEGER,
    EXP_NR      INTEGER,
    NEW_USER    INTEGER,
    NEW_ZEIT    TEXT,
    CHG_USER    INTEGER,
    CHG_ZEIT    TEXT
);

-- =============================================================
-- DAILYTOTALS1  (totales diarios)
-- =============================================================
CREATE TABLE DAILYTOTALS1 (
    KST_ID          INTEGER NOT NULL REFERENCES KOSTST(KST_ID),
    ART_ID          INTEGER NOT NULL REFERENCES ARTIKEL(ART_ID),
    DAY_DATE        TEXT NOT NULL,
    DAY_SOHBEG      REAL,
    DAY_SOHEND      REAL,
    DAY_QTYPURCH    INTEGER,
    DAY_QTYTRSFIN   INTEGER,
    DAY_QTYTRSFOUT  INTEGER,
    DAY_QTYUSAGE    INTEGER,
    DAY_QTYSOLD     INTEGER,
    DAY_QTYINV      INTEGER,
    DAY_SOHINV      INTEGER,
    DAY_INVDATE     TEXT,
    DAY_PRICELAST   REAL,
    DAY_PRICEAVG    REAL,
    DAY_PRICELPURCH REAL,
    DAY_PRICE       REAL,
    DAY_PRICESTD    REAL,
    DAY_SUMPURCH    REAL,
    PRIMARY KEY (KST_ID, ART_ID, DAY_DATE)
);

-- =============================================================
-- Índices recomendados
-- =============================================================
CREATE INDEX idx_lfs_lf      ON LIEFERSCHEIN(LF_ID);
CREATE INDEX idx_lfs_status  ON LIEFERSCHEIN(LFS_STATUS);
CREATE INDEX idx_lfs_rts     ON LIEFERSCHEIN(LFS_RTS);
CREATE INDEX idx_lfs_datum   ON LIEFERSCHEIN(LFS_DATUM);

CREATE INDEX idx_lfp_artnr   ON LIEFERPOS(ART_NR);
CREATE INDEX idx_lfp_kst     ON LIEFERPOS(KST_ID);

CREATE INDEX idx_art_wgr     ON ARTIKEL(WGR_ID);
CREATE INDEX idx_art_nummer  ON ARTIKEL(ART_NUMMER);

CREATE INDEX idx_inv_kst     ON INVENTUR(KST_ID);
CREATE INDEX idx_inv_status  ON INVENTUR(INV_STATUS);

CREATE INDEX idx_invp_art    ON INVPOSART(ART_ID);
CREATE INDEX idx_invp_inv    ON INVPOSART(INV_ID);

CREATE INDEX idx_dt_date     ON DAILYTOTALS1(DAY_DATE);
```

---

## 6. Consideraciones para el código Go

### 6.1. Orden de carga (importa por FKs)

```
1. KOSTST           (sin dependencias estrictas; KST_PARENT no es FK estricta)
2. LIEFER           (sin dependencias)
3. WARENGRUPPE      (sin dependencias)
4. VPCKEINH         (sin dependencias)
5. ARTIKEL          (depende de WARENGRUPPE, VPCKEINH)
6. LIEFERSCHEIN     (depende de LIEFER)
7. LIEFERPOS        (depende de LIEFERSCHEIN, ARTIKEL, VPCKEINH, KOSTST, LIEFER)
8. INVENTUR         (depende de KOSTST)
9. INVPOSART        (depende de ARTIKEL, VPCKEINH, WARENGRUPPE; INV_ID NO es FK estricta)
10. HIS_VERBRAUCH   (depende de KOSTST)
11. DAILYTOTALS1    (depende de KOSTST, ARTIKEL)
```

### 6.2. Manejo de FKs en SQLite

- SQLite no fuerza FKs por defecto. Activar con `PRAGMA foreign_keys = ON;`.
- Recomendación: **dejar `OFF` durante la carga** y **activar al final**, o validar manualmente con queries.
- **No declarar FK estricta** en `KOSTST.KST_PARENT` ni en `INVPOSART.INV_ID` (datos huérfanos esperados en este export).

### 6.3. Performance de inserción

```
PRAGMA journal_mode = WAL;
PRAGMA synchronous  = NORMAL;
PRAGMA foreign_keys = OFF;
BEGIN TRANSACTION;
-- bulk inserts con prepared statements
COMMIT;
```

### 6.4. Conversión de tipos

| Valor en CSV | Conversión |
|---|---|
| `""` (vacío) | `NULL` |
| `"123"` | `int64` para columnas INTEGER |
| `"1009.7500"` | `float64` para columnas REAL |
| `"-303968.524"` | `float64` (negativos posibles en NCs/devoluciones) |
| `"2026-04-23 00:00:00"` | dejar como string (TEXT) |

### 6.5. Encoding y separador

- Todos los archivos están en **UTF-8** y usan **coma `,`** como separador.
- El código puede usar `encoding/csv` directo con la configuración por defecto.

### 6.6. Estructura sugerida del programa

```
main.go
├─ openDB(path)             // crea SQLite, ejecuta DDL
├─ loadCSV(path) → []row    // genérico: lee CSV, devuelve slice de maps o structs
├─ insertBatch(table, rows) // genérico: inserta en transacción con prepared statement
└─ main()
    ├─ openDB
    ├─ ejecutarDDL
    ├─ por cada archivo CSV en orden de dependencias:
    │   ├─ loadCSV
    │   └─ insertBatch
    ├─ activarFKs
    └─ validarReferencias  // queries de control
```

### 6.7. Bibliotecas Go recomendadas

| Necesidad | Biblioteca |
|---|---|
| CSV | `encoding/csv` (stdlib) |
| SQLite (puro Go, sin CGO) | `modernc.org/sqlite` |
| SQLite (con CGO, más rápido) | `github.com/mattn/go-sqlite3` |
| Logging | `log/slog` (stdlib) |

### 6.8. Mapeo nombre-archivo → nombre-tabla

| Archivo | Tabla |
|---|---|
| `Kostst_*.csv` | `KOSTST` |
| `Liefer_*.csv` | `LIEFER` |
| `Warengruppe_*.csv` | `WARENGRUPPE` |
| `Vpckeinh_*.csv` | `VPCKEINH` |
| `Artikel_*.csv` | `ARTIKEL` |
| `Lieferschein_*.csv` | `LIEFERSCHEIN` |
| `Lieferpos_*.csv` | `LIEFERPOS` |
| `Inventur_*.csv` | `INVENTUR` |
| `Invposart_*.csv` | `INVPOSART` |
| `His_verbrauch_*.csv` | `HIS_VERBRAUCH` |
| `Dailytotals_*.csv` | `DAILYTOTALS1` |

> El sufijo `_YYYYMMDD` corresponde a la fecha del export.

---

## 7. Validaciones post-carga

```sql
-- Conteos por tabla
SELECT 'KOSTST' AS tabla, COUNT(*) AS filas FROM KOSTST
UNION ALL SELECT 'LIEFER', COUNT(*) FROM LIEFER
UNION ALL SELECT 'WARENGRUPPE', COUNT(*) FROM WARENGRUPPE
UNION ALL SELECT 'VPCKEINH', COUNT(*) FROM VPCKEINH
UNION ALL SELECT 'ARTIKEL', COUNT(*) FROM ARTIKEL
UNION ALL SELECT 'LIEFERSCHEIN', COUNT(*) FROM LIEFERSCHEIN
UNION ALL SELECT 'LIEFERPOS', COUNT(*) FROM LIEFERPOS
UNION ALL SELECT 'INVENTUR', COUNT(*) FROM INVENTUR
UNION ALL SELECT 'INVPOSART', COUNT(*) FROM INVPOSART
UNION ALL SELECT 'HIS_VERBRAUCH', COUNT(*) FROM HIS_VERBRAUCH
UNION ALL SELECT 'DAILYTOTALS1', COUNT(*) FROM DAILYTOTALS1;

-- Filas esperadas (snapshot 2026-05-05):
-- KOSTST:        2
-- LIEFER:        54
-- WARENGRUPPE:   627
-- VPCKEINH:      73
-- ARTIKEL:       437
-- LIEFERSCHEIN:  12
-- LIEFERPOS:     150
-- INVENTUR:      1
-- INVPOSART:     1022
-- HIS_VERBRAUCH: 1
-- DAILYTOTALS1:  438

-- FKs huérfanas (deberían dar 0 excepto la marcada)
SELECT 'LIEFERPOS sin LIEFERSCHEIN' AS check_, COUNT(*) AS huerfanos
FROM LIEFERPOS p LEFT JOIN LIEFERSCHEIN c ON p.LFS_ID = c.LFS_ID
WHERE c.LFS_ID IS NULL
UNION ALL
SELECT 'LIEFERPOS sin ARTIKEL', COUNT(*)
FROM LIEFERPOS p LEFT JOIN ARTIKEL a ON p.ART_NR = a.ART_ID
WHERE a.ART_ID IS NULL
UNION ALL
SELECT 'INVPOSART sin INVENTUR (esperado >0)', COUNT(*)
FROM INVPOSART p LEFT JOIN INVENTUR i ON p.INV_ID = i.INV_ID
WHERE i.INV_ID IS NULL
UNION ALL
SELECT 'DAILYTOTALS1 sin ARTIKEL', COUNT(*)
FROM DAILYTOTALS1 d LEFT JOIN ARTIKEL a ON d.ART_ID = a.ART_ID
WHERE a.ART_ID IS NULL;
```
