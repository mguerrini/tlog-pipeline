# SEQUENCENUMBER — Algoritmo de construcción y descomposición

Documento de referencia técnica del campo `SEQUENCENUMBER` usado en los TLOG OCPRA.

---

## 1. Resumen del formato

El `SEQUENCENUMBER` es un identificador numérico de **12 dígitos** que identifica de manera única cada documento TLOG por **(día × tipo × contador)**.

```
Posiciones:  1  2 3  4 5  6 7  8   9 10 11 12
             9  A A  M M  D D  D   C  C  C  C
             │  └AA┘ └MM┘ └DD┘ │   └──counter──┘
             │                 │
             │                 └ DOC (0..7)
             └ prefijo fijo
```

| Tramo | Long | Contenido |
|---|---|---|
| `9` | 1 | Prefijo fijo. |
| `AAMMDD` | 6 | Año (2 díg) + mes (2) + día (2) del `BusinessDayDate`. |
| `DOC` | 1 | Índice del tipo de archivo TLOG (0..7). |
| `CONTADOR` | 4 | Secuencial por (día × tipo) que arranca en 0 y se incrementa para cada documento adicional del mismo tipo en el mismo día. Rango 0..9999. |

### Tipos de documento

| Idx | TLOG |
|---|---|
| 0 | Reception |
| 1 | Return |
| 2 | Transfer |
| 3 | Adjustment |
| 4 | Count |
| 5 | FiscalDocFC |
| 6 | FiscalDocNC |
| 7 | Cierre |

---

## 2. Algoritmo de construcción

```
Entradas:
  businessDayDate   : fecha del archivo (date)
  documentNumber    : entero 0..7
  counter           : entero 0..9999 (índice del documento dentro del día/tipo)

Salida:
  seq               : string de 12 dígitos

Pasos:
  1. Validar que documentNumber ∈ [0..7]   sino → ERROR
  2. Validar que counter        ∈ [0..9999] sino → ERROR
  3. aa = año(businessDayDate) mod 100
     mm = mes(businessDayDate)
     dd = día(businessDayDate)
  4. seq = "9"
         + sprintf("%02d", aa)
         + sprintf("%02d", mm)
         + sprintf("%02d", dd)
         + sprintf("%d",   documentNumber)
         + sprintf("%04d", counter)
  5. return seq
```

### 2.1. Ejemplo paso a paso

Datos: `businessDayDate=2026-10-27`, `documentNumber=0` (Reception), `counter=0`.

1. `documentNumber=0` ∈ [0,7] → OK.
2. `counter=0` ∈ [0,9999] → OK.
3. `aa=26`, `mm=10`, `dd=27`.
4. `seq = "9" + "26" + "10" + "27" + "0" + "0000" = "926102700000"`.

### 2.2. Más ejemplos

| Fecha | DOC | Counter | SEQ |
|---|---|---|---|
| 2026-01-01 | 0 (Reception)  | 0 | `926010100000` |
| 2026-01-01 | 0 (Reception)  | 1 | `926010100001` |
| 2026-01-01 | 6 (FiscalDocNC)| 0 | `926010160000` |
| 2026-01-01 | 7 (Cierre)     | 0 | `926010170000` |
| 2026-10-27 | 3 (Adjustment) | 0 | `926102730000` |
| 2026-10-27 | 3 (Adjustment) | 5 | `926102730005` |

---

## 3. Algoritmo inverso (decodificación)

```
Entrada:
  seq               : string numérico de 12 dígitos
  yearWindowStart   : año base para resolver AA (ej. 2000)

Salida:
  año, mes, día, documentNumber, counter

Pasos:
  1. Validar que seq sea solo dígitos. Sino → ERROR.
  2. Validar len(seq)==12. Sino → ERROR.
  3. Validar seq[0]=='9'. Sino → ERROR.
  4. aa             = parseInt(seq[1..3))
     mm             = parseInt(seq[3..5))
     dd             = parseInt(seq[5..7))
     documentNumber = parseInt(seq[7..8))
     counter        = parseInt(seq[8..12))
  5. Validar documentNumber ∈ [0..7].
  6. año = expandirAA(aa, yearWindowStart)
  7. Validar que (año, mm, dd) sea una fecha calendario válida.
```

### 3.1. Resolución de `AA` → año completo (`expandirAA`)

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

---

## 4. Garantías matemáticas

| Propiedad | Verificación |
|---|---|
| **Único por (fecha × tipo × contador)** | Cada tramo aporta unicidad: dos `SEQUENCENUMBER` distintos solo si difieren en al menos uno de día / mes / año / tipo / contador. |
| **Monotónico creciente dentro de (día, tipo)** | El contador se incrementa estrictamente (0, 1, 2, …) por cada nuevo documento del mismo tipo en el mismo día. |
| **Reversible** | Cada tramo es extraíble por posición. |

### Limitaciones conocidas

- **Cambio de siglo:** el tramo `AA` de 2 dígitos puede colisionar cada 100 años (ej. 2026-01-01 vs 2126-01-01). Mitigante: `Decode` acepta `yearWindowStart` para fijar la ventana.
- **Capacidad del contador:** 4 dígitos = máximo 10 000 documentos del mismo tipo en el mismo día. Si se excede, `Build` retorna error.

---

## 5. API Go (`package sequence`)

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

type Decoded struct {
    Year           int
    Month          int
    Day            int
    DocumentNumber DocumentNumber
    Counter        int
}
```

### Funciones

```go
// Build construye el SEQUENCENUMBER con formato 9AAMMDD<DOC><CONTADOR4>.
// counter arranca en 0 para el primer documento del tipo en el día.
func Build(businessDayDate time.Time, doc DocumentNumber, counter int) (string, error)

// Decode descompone un SEQUENCENUMBER en sus componentes lógicos.
// yearWindowStart resuelve la ambigüedad del tramo AA (usar 2000 para siglo XXI).
func Decode(seq string, yearWindowStart int) (Decoded, error)

// BusinessDayDate reconstruye la fecha (medianoche UTC).
func (d Decoded) BusinessDayDate() time.Time
```

### Errores devueltos

`Build` retorna error si:
- `documentNumber` está fuera de `[0, 7]`.
- `counter` está fuera de `[0, 9999]`.

`Decode` retorna error si:
- El string contiene caracteres no numéricos.
- La longitud no es 12.
- El primer dígito no es `9`.
- El `documentNumber` decodificado está fuera de `[0, 7]`.
- La fecha decodificada no es un día calendario válido.

---

## 6. Ejemplo de uso (Go)

```go
package main

import (
    "fmt"
    "time"

    "github.com/opessa/tlog-pipeline/internal/sequence"
)

func main() {
    day := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)

    // Primer documento Adjustment del día.
    seq, _ := sequence.Build(day, sequence.DocAdjustment, 0)
    fmt.Println(seq) // 926102730000

    // Segundo documento Adjustment del mismo día.
    seq2, _ := sequence.Build(day, sequence.DocAdjustment, 1)
    fmt.Println(seq2) // 926102730001

    // Decodificar.
    dec, _ := sequence.Decode(seq2, 2000)
    fmt.Printf("año=%d mes=%d día=%d doc=%s counter=%d\n",
        dec.Year, dec.Month, dec.Day, dec.DocumentNumber, dec.Counter)
    // año=2026 mes=10 día=27 doc=Adjustment counter=1
}
```
