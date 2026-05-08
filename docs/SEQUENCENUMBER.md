# SEQUENCENUMBER — Algoritmo de construcción y descomposición

Documento de referencia técnica del campo `SEQUENCENUMBER` usado en los TLOG OCPRA.

---

## 1. Resumen del formato

El `SEQUENCENUMBER` es un identificador numérico único por **(retail × año × día × archivo TLOG)**, sin saltos numéricos dentro del año, monotónicamente creciente.

Tiene dos longitudes según el origen del TLOG:

| Origen | Longitud | TLOGs que lo usan |
|---|---|---|
| **SU** | **12** dígitos | Reception, Return, Transfer, Adjustment, Count, FiscalDocFC, FiscalDocNC |
| **Bridge** | **11** dígitos | Cierre |

Composición:

```
SU      (12): [RETAILID:6][AA:2][NUMERO_ARCHIVO:4]
Bridge  (11): [RETAILID:5][AA:2][NUMERO_ARCHIVO:4]
```

---

## 2. Definición de los tramos

| Tramo | Long (SU / Bridge) | Contenido | Origen |
|---|---|---|---|
| `RETAILID` | 6 / 5 | `KST_CODE` rellenado con ceros a izquierda hasta llenar el ancho del tramo | `KOSTST.KST_CODE` |
| `AA` | 2 | Últimos 2 dígitos del año de `BusinessDayDate` (ej. 2026 → `26`) | nombre del archivo (`AAAAMMDD`) |
| `NUMERO_ARCHIVO` | 4 | `(DDD - 1) * 8 + DOCUMENT_NUMBER` | calculado |

Donde:

- `DDD` = día del año, valor entre **1 y 366** (366 solo en años bisiestos).
- `DOCUMENT_NUMBER` = índice del tipo de archivo TLOG, valor entre **0 y 7**:

| Idx | TLOG | Origen |
|---|---|---|
| 0 | Reception | SU |
| 1 | Return | SU |
| 2 | Transfer | SU |
| 3 | Adjustment | SU |
| 4 | Count | SU |
| 5 | FiscalDocFC | SU |
| 6 | FiscalDocNC | SU |
| 7 | Cierre | Bridge |

> **Por qué `(DDD-1)*8 + DOCUMENT_NUMBER` y no `[DDD:3][DOC:1]`:** un dígito decimal tiene base 10, pero solo hay 8 valores válidos para `DOCUMENT_NUMBER` (0..7). Si se concatenan tramos separados, cada cambio de día deja sin usar los valores 8 y 9 del último dígito → se generan **2 saltos diarios**. El cálculo `(DDD-1)*8 + DOC` empaqueta los 8 valores válidos de forma contigua dentro del tramo de 4 dígitos, sin huecos.

**Capacidad del tramo `NUMERO_ARCHIVO` (4 dígitos):**
- Rango usado: 0..2927 (año bisiesto: `(366-1)*8 + 7 = 2927`).
- Rango disponible: 0..9999 → margen amplio.

---

## 3. Algoritmo de construcción

```
Entradas:
  retailID          : string numérico (KST_CODE)
  businessDayDate   : fecha del archivo (date)
  documentNumber    : entero 0..7

Salida:
  seq               : string de 11 ó 12 dígitos

Pasos:
  1. Si documentNumber == 7 (Cierre)   → origin = Bridge, retailWidth = 5
     Si documentNumber  ∈ [0..6]       → origin = SU,     retailWidth = 6
     En cualquier otro caso            → ERROR

  2. Validar que retailID sea numérico y su longitud ≤ retailWidth.
     Sino                              → ERROR

  3. retailPadded = retailID con ceros a izquierda hasta retailWidth

  4. aa  = año(businessDayDate) mod 100        // 2 dígitos
     ddd = díaDelAño(businessDayDate)          // 1..366

  5. numeroArchivo = (ddd - 1) * 8 + documentNumber

  6. seq = retailPadded
         + sprintf("%02d",   aa)
         + sprintf("%04d",   numeroArchivo)

  7. return seq
```

### 3.1. Ejemplo paso a paso

Datos: `retailID="00001"`, `businessDayDate=2026-10-27`, `documentNumber=0` (Reception).

1. `documentNumber=0` → origin = SU, retailWidth = 6.
2. `"00001"` es numérico y len=5 ≤ 6 → OK.
3. `retailPadded = "000001"` (un cero más a izquierda).
4. `aa = 2026 mod 100 = 26`. `ddd = 300` (27-oct-2026 es el día 300 del año).
5. `numeroArchivo = (300 - 1) * 8 + 0 = 2392`.
6. `seq = "000001" + "26" + "2392" = "000001262392"`.

---

## 4. Algoritmo inverso (decodificación)

```
Entrada:
  seq               : string numérico
  yearWindowStart   : año base para resolver AA (ej. 2000)

Salida:
  retailID, año, día_del_año, documentNumber

Pasos:
  1. Validar que seq sea solo dígitos. Sino → ERROR.

  2. Según len(seq):
       12 → origin = SU,     retailWidth = 6
       11 → origin = Bridge, retailWidth = 5
       otro → ERROR

  3. retailID      = seq[0 .. retailWidth)
     aa            = parseInt(seq[retailWidth .. retailWidth+2))
     numeroArchivo = parseInt(seq[retailWidth+2 .. fin))

  4. dayOfYear      = numeroArchivo / 8 + 1     (división entera)
     documentNumber = numeroArchivo mod 8

  5. Validar consistencia origen ↔ documentNumber:
       Si documentNumber==7 y origin!=Bridge → ERROR
       Si documentNumber!=7 y origin!=SU     → ERROR

  6. año = expandirAA(aa, yearWindowStart)

  7. Validar dayOfYear ∈ [1, 366]
     Si dayOfYear == 366 y año no es bisiesto → ERROR
```

### 4.1. Resolución de `AA` → año completo (`expandirAA`)

El tramo `AA` tiene solo 2 dígitos, por lo que es ambiguo en horizontes >100 años. Se resuelve con una **ventana móvil**:

```
expandirAA(aa, yearWindowStart):
  si yearWindowStart == 0:
    return 2000 + aa                          // default siglo XXI

  startCentury = (yearWindowStart / 100) * 100
  candidate    = startCentury + aa
  si candidate < yearWindowStart:
    candidate += 100
  return candidate
```

Ejemplos con `yearWindowStart=2000`:

| AA | año resultante |
|---|---|
| 26 | 2026 |
| 99 | 2099 |
| 00 | 2000 |

Ejemplos con `yearWindowStart=2010`:

| AA | año resultante | razón |
|---|---|---|
| 26 | 2026 | 2026 ≥ 2010 |
| 05 | 2105 | 2005 < 2010, rota a 2105 |

### 4.2. Ejemplo paso a paso

Datos: `seq="000001262392"`, `yearWindowStart=2000`.

1. Solo dígitos → OK.
2. `len=12` → origin=SU, retailWidth=6.
3. `retailID="000001"`, `aa=26`, `numeroArchivo=2392`.
4. `dayOfYear = 2392 / 8 + 1 = 299 + 1 = 300`. `documentNumber = 2392 mod 8 = 0` → Reception.
5. `documentNumber=0`, origin=SU → consistente.
6. `año = 2000 + 26 = 2026`.
7. `dayOfYear=300` válido (≤365 y no es 366) → OK.

Resultado: `retail=000001`, `2026-10-27` (día 300), `Reception`.

---

## 5. Análisis de bordes

### 5.1. Cambio de documento (mismo día)

`KST=00001`, `año=2026`, día 1 (1-ene-2026):

| DOC | NUMERO_ARCHIVO | SEQ |
|---|---|---|
| 0 (Reception) | 0 | `000001260000` |
| 1 (Return) | 1 | `000001260001` |
| 2 (Transfer) | 2 | `000001260002` |
| 3 (Adjustment) | 3 | `000001260003` |
| 4 (Count) | 4 | `000001260004` |
| 5 (FiscalDocFC) | 5 | `000001260005` |
| 6 (FiscalDocNC) | 6 | `000001260006` |
| 7 (Cierre) | 7 | `00001260007` (11 díg) |

✅ Sin saltos.

### 5.2. Cambio de día

`KST=00001`, año 2026, transición día 1 → día 2:

| Fecha | DOC | NUMERO_ARCHIVO | SEQ |
|---|---|---|---|
| 1-ene | 7 (Cierre) | 7 | `00001260007` |
| 2-ene | 0 (Reception) | 8 | `000001260008` |
| 2-ene | 1 (Return) | 9 | `000001260009` |
| 2-ene | 2 (Transfer) | 10 | `000001260010` |

✅ Secuencia 7, 8, 9, 10... contigua.

### 5.3. Último día de año no bisiesto

`KST=00001`, año 2026, día 365 (31-dic):

| DOC | NUMERO_ARCHIVO | SEQ |
|---|---|---|
| 0 | 2912 | `000001262912` |
| 7 | 2919 | `00001262919` |

Tramo `NUMERO_ARCHIVO` cabe holgadamente (max 9999, uso 2919).

### 5.4. Cambio de año

| Fecha | DOC | SEQ |
|---|---|---|
| 31-dic-2026 | 7 (Cierre) | `00001262919` |
| 1-ene-2027 | 0 (Reception) | `000001270000` |

✅ El reset de `NUMERO_ARCHIVO` (2919 → 0) **no genera colisión** porque el tramo `AA` cambió (`26` → `27`). Numéricamente: `000001262919 < 000001270000` → orden monotónico preservado.

### 5.5. Año bisiesto (366 días)

2028 es bisiesto. Día 366 (31-dic-2028), `KST=00001`:

| DOC | NUMERO_ARCHIVO | SEQ |
|---|---|---|
| 0 | (366-1)·8+0 = 2920 | `000001282920` |
| 7 | (366-1)·8+7 = **2927** | `00001282927` |

**Pico histórico de uso: 2927.** Sigue dentro del rango de 4 dígitos.

### 5.6. Cambio de siglo (limitación conocida)

⚠️ **Limitación inherente al uso de AA de 2 dígitos:** cada 100 años puede haber colisión potencial.

| Fecha | AA | SEQ |
|---|---|---|
| 1-ene-2026 | 26 | `000001260000` |
| 1-ene-2126 | 26 | `000001260000` ← idéntico |

**Mitigantes:**

- Horizonte real esperado del sistema: típicamente <50 años → la colisión nunca ocurre en la práctica.
- Si se necesita blindar a >100 años: cambiar `AA` por `AAAA` (4 dígitos). Esto reduce el ancho disponible para `RETAILID`:
  - SU: `[RETAILID:4][AAAA:4][NUMERO_ARCHIVO:4]` → max 9999 retails.
  - Bridge: `[RETAILID:3][AAAA:4][NUMERO_ARCHIVO:4]` → max 999 retails.

---

## 6. Garantías matemáticas

| Propiedad | Verificación |
|---|---|
| **Único** por (retail × año × día × archivo) | Cada tramo aporta unicidad: dos `SEQUENCENUMBER` distintos solo si difieren en al menos uno de retail / año / día / archivo. |
| **Sin saltos** dentro del año | Para un retail dado en un año fijo, los `NUMERO_ARCHIVO` recorren `0, 1, 2, …, 8·DiasDelAño - 1` sin huecos (verificado por test `TestNoGaps_FullYear`). |
| **Monotónico creciente** en el tiempo | Mayor fecha → mayor `NUMERO_ARCHIVO` dentro del año; mayor año → mayor `AA` (válido por 100 años). |
| **Reversible** (algoritmo inverso) | Cada tramo es extraíble por posición; `NUMERO_ARCHIVO` se descompone unívocamente: `DDD = NUMERO_ARCHIVO/8 + 1`, `DOC = NUMERO_ARCHIVO mod 8`. |

---

## 7. API Go (`package sequence`)

### Tipos

```go
type DocumentNumber int
const (
    DocReception   DocumentNumber = 0
    DocReturn      DocumentNumber = 1
    DocTransfer    DocumentNumber = 2
    DocAdjustment  DocumentNumber = 3
    DocCount       DocumentNumber = 4
    DocFiscalDocFC DocumentNumber = 5
    DocFiscalDocNC DocumentNumber = 6
    DocCierre      DocumentNumber = 7
)

type Origin int
const (
    OriginSU     Origin = iota // 12 dígitos
    OriginBridge               // 11 dígitos
)

type Decoded struct {
    Origin         Origin
    RetailID       string
    Year           int
    DayOfYear      int
    DocumentNumber DocumentNumber
    NumeroArchivo  int
}
```

### Funciones

```go
// Build construye el SEQUENCENUMBER. El origen (SU vs Bridge) se infiere
// del DocumentNumber: Cierre→Bridge, resto→SU.
func Build(retailID string, businessDayDate time.Time, doc DocumentNumber) (string, error)

// Decode descompone un SEQUENCENUMBER en sus componentes lógicos.
// yearWindowStart resuelve la ambigüedad del tramo AA (usar 2000 para siglo XXI).
func Decode(seq string, yearWindowStart int) (Decoded, error)

// BusinessDayDate reconstruye la fecha (medianoche UTC).
func (d Decoded) BusinessDayDate() time.Time
```

### Errores devueltos

`Build` retorna error si:
- `documentNumber` está fuera de `[0, 7]`.
- `retailID` no es numérico o excede el ancho del tramo.

`Decode` retorna error si:
- El string contiene caracteres no numéricos.
- La longitud no es 11 ni 12.
- El `documentNumber` decodificado no concuerda con el origen (ej: 12 dígitos con DOC=7).
- `dayOfYear` está fuera de [1, 366].
- `dayOfYear=366` en año no bisiesto.

---

## 8. Ejemplos de uso (Go)

```go
package main

import (
    "fmt"
    "time"

    "github.com/foody/sequence"
)

func main() {
    // Construir SEQ
    day := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC) // día 300
    seq, err := sequence.Build("00001", day, sequence.DocReception)
    if err != nil {
        panic(err)
    }
    fmt.Println(seq) // 000001262392

    // Decodificar SEQ
    dec, err := sequence.Decode(seq, 2000)
    if err != nil {
        panic(err)
    }
    fmt.Printf("retail=%s año=%d día=%d doc=%s\n",
        dec.RetailID, dec.Year, dec.DayOfYear, dec.DocumentNumber)
    // retail=000001 año=2026 día=300 doc=Reception

    fmt.Println(dec.BusinessDayDate().Format("2006-01-02"))
    // 2026-10-27
}
```

---

## 9. Verificación de tests

El paquete incluye 13 tests que cubren:

- Casos de ejemplo del enunciado (KST=00001, día 300).
- Cambio de documento (sin saltos dentro del día).
- Cambio de día (continuidad numérica).
- Último día de año no bisiesto y bisiesto.
- Cambio de año (sin colisiones por cambio de AA).
- Padding de RETAILID en SU y Bridge.
- Errores de Build (retailID inválido, doc fuera de rango).
- Roundtrip Build → Decode preservando todos los componentes.
- Consistencia origen ↔ DocumentNumber.
- Longitudes inválidas en Decode.
- Caracteres no numéricos en Decode.
- Resolución de AA con distintas ventanas (`yearWindowStart`).
- Validación de DDD=366 en años no bisiestos.
- **Año completo sin saltos**: 2920 SEQs consecutivos verificados (365 días × 8 docs).

Ejecutar con:

```bash
go test -v ./...
```
