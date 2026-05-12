# Manual de configuración — tlog-pipeline

Guía rápida para usar `config.json` y los flags de CLI.

## Cómo se ejecuta

```
pipeline --config ./config.json
```

Sin flags, el pipeline procesa **todos los días** que encuentre en `local_folders.source_root` (sub-carpetas `AAAAMMDD/`) y/o en `local_folders.all` (CSVs cuyo nombre termina en `AAAAMMDD`).

## Estructura de carpetas

| Carpeta | Para qué sirve |
|---|---|
| `local_folders.all` | Pozo único donde caen los CSVs sin separar por día (ej. desde un FTP/SFTP). El step `split_by_date` los reparte. |
| `local_folders.source_root` | Carpeta de input por día — contiene sub-carpetas `AAAAMMDD/` con los CSVs de ese día. |
| `local_folders.target_root` | Carpeta de output — el pipeline genera ahí la base SQLite, los XMLs y los logs por día. |
| `local_folders.finished_root` | Destino final cuando `local_clean` mueve lo procesado. |

## Secciones de `config.json`

### `local_folders`
Las cuatro rutas de arriba. **Obligatorio** definir `source_root` y `target_root`.

### `ftp_folders`
Rutas **remotas** que usan los steps `ftp_*`. Centralizadas para que ningún step las repita.

| Campo | Lo usa |
|---|---|
| `source_root` | `ftp_download` (de dónde bajar). |
| `target_root` | `ftp_upload` (a dónde subir los XMLs). |
| `finished_root` | `ftp_end` (a dónde mover los CSVs ya procesados). |

### `process`
Cómo se procesan los días.

- `mode`: `ALL` (todos los días) o `DAY` (un día — usar con `--day`).
- `execution_mode`: `PARALLEL` (varios días en paralelo) o `SERIAL` (uno por uno).
- `parallel_retails_per_day`: si `true`, fuerza modo serial entre días pero permite paralelismo dentro del día.
- `begin_date_offset` / `end_date_offset`: ventana horaria del día (formato `HH:MM:SS`). Define qué horas pertenecen a qué día (ej. cierres de turno que cruzan medianoche).
- `operator_id`: ID del operador que se escribe en los TLOGs.

### `output`
Qué TLOGs (XMLs) se generan. Cada flag activa o desactiva un tipo:

```
cierre, inventory_reception, inventory_fiscaldoc_fc, inventory_fiscaldoc_nc,
inventory_return, inventory_adjustment, inventory_count, inventory_transfer
```

> Si la sección se **omite**, se generan los 8. Si está presente y un flag falta, ese flag se asume `true`.

### `logs`
Activa la escritura de archivos de log/reporte por día (todos opcionales):

- `pipeline_enabled` → `AAAAMMDD_pipeline.log`
- `day_status_enabled` → `AAAAMMDD_day_status.json`
- `sql_db_load` → `AAAAMMDD_sqldb_load.md`
- `orphans_report` → `AAAAMMDD_orphans.md`

> Misma regla de defaults que `output`: omitir la sección equivale a `true` en todos.

### Steps del pipeline (orden de ejecución)

Cada step tiene su propia sección con al menos `enabled`. Si `enabled: false` el step se saltea.

| Step | Flags propios | Qué hace |
|---|---|---|
| `ftp_download` | `enabled`, `split_by_date` | Baja CSVs desde `ftp_folders.source_root`. Si `split_by_date: true`, asume los archivos sueltos en remoto y los deja en `local_folders.all` (los reparte el step `split_by_date` local). Si `false`, asume sub-carpetas remotas con fecha (`AAAAMMDD` o `AAAA-MM-DD`) y baja a `local_folders.source_root/AAAAMMDD/`. |
| `split_by_date` | `enabled` | Mueve los CSVs de `all/` a `source_root/AAAAMMDD/` según los últimos 8 dígitos del nombre. |
| `read_files` | `enabled`, `expected_files` | Valida que estén los CSVs esperados en la carpeta del día. |
| `create_db` | `enabled`, `separator`, `sql` | Carga los CSVs en una base. Si `sql: true`, genera además un SQLite tipado (modo debug — el pipeline termina ahí). El `separator` es el delimitador de los CSVs de entrada (típicamente `|`). |
| `create_xml` / `create_xml_sql` | `enabled` | Generan los TLOGs XML según los flags de `output`. |
| `ftp_upload` | `enabled` | Sube los XMLs a `ftp_folders.target_root`. |
| `ftp_end` | `enabled`, `delete_local_source` | Archiva los CSVs procesados moviéndolos en remoto a `ftp_folders.finished_root`. Con `delete_local_source: true` borra además la carpeta local `target_root/AAAAMMDD/` entera. |
| `local_clean` | `enabled`, `delete_source`, `delete_database` | Limpia las carpetas locales. Con `delete_source: true` borra los CSVs de input; con `delete_database: true` borra la SQLite. |

> Las rutas que usa cada step ya no se configuran por step — vienen de `local_folders` (rutas locales) y `ftp_folders` (rutas remotas).

### `ftp_source` / `ftp_target`
Credenciales FTP/SFTP (`url`, `port`, `username`, `pass`). Se usan en los steps `ftp_*`. `port` típicamente `22` para SFTP, `21` para FTP plano.

### `read_files.expected_files`
Lista de patrones glob que deben existir en la carpeta del día. Si falta alguno, el step falla. Si la lista se omite, se usa la default (los 11 CSVs del proyecto).

## Flags de CLI

| Flag | Efecto |
|---|---|
| `--config <path>` | Path al `config.json` (default `./config.json`). |
| `--day AAAAMMDD` | Procesa solo ese día (acepta `AAAA-MM-DD`). |
| `--step <nombre>` | Ejecuta solo ese step. **Requiere `--day`**. |
| `--folder-source-root <path>` | Override de `local_folders.source_root`. |
| `--folder-target-root <path>` | Override de `local_folders.target_root`. |
| `--folder-finished <path>` | Override de `local_folders.finished_root`. |
| `--ftp-disabled` | Desactiva los 3 steps de FTP. |
| `--delete-source` | Fuerza `local_clean.delete_source = true`. |

## Escenarios típicos

**Procesar todos los días disponibles, sin tocar FTP:**
```
pipeline --ftp-disabled
```

**Procesar un día puntual:**
```
pipeline --day 20260510
```

**Re-correr un solo step de un día:**
```
pipeline --day 20260510 --step create_xml
```

**Modo debug (generar SQLite y parar):**
En `config.json`, dejar `create_db.enabled: true` y `create_db.sql: true`. El pipeline termina después de generar el `.db`.

**Pipeline solo local (sin FTP):**
Poner `enabled: false` en `ftp_download`, `ftp_upload`, `ftp_end`. Los CSVs se asumen en `local_folders.all` o ya repartidos en `source_root/AAAAMMDD/`.
