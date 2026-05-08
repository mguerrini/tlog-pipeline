package config

func applyDefaults(c *Config) {
	if c.Process.Mode == "" {
		c.Process.Mode = "ALL"
	}
	if c.Process.ExecutionMode == "" {
		c.Process.ExecutionMode = "PARALLEL"
	}
	if c.Process.BeginDateOffset == "" {
		c.Process.BeginDateOffset = "00:00:00"
	}
	if c.Process.EndDateOffset == "" {
		c.Process.EndDateOffset = "23:59:59"
	}
	if c.Process.OperatorID == "" {
		c.Process.OperatorID = "admin"
	}
	if c.CreateDB.Separator == "" {
		c.CreateDB.Separator = ","
	}
	// Sección "logs" omitida del JSON → habilitar todos (compat retro).
	// Si está presente, UnmarshalJSON ya resolvió cada flag (false explícito
	// se respeta; flags ausentes default-ean a true).
	if c.Logs == nil {
		c.Logs = &Logs{
			PipelineEnabled:  true,
			DayStatusEnabled: true,
			SQLDBLoad:        true,
			OrphansReport:    true,
		}
	}
	// Sección "output" omitida del JSON → generar los 8 TLOGs (compat retro).
	if c.Output == nil {
		c.Output = &Output{
			Cierre:               true,
			InventoryReception:   true,
			InventoryFiscalDocFC: true,
			InventoryFiscalDocNC: true,
			InventoryReturn:      true,
			InventoryAdjustment:  true,
			InventoryCount:       true,
			InventoryTransfer:    true,
		}
	}
	if len(c.ReadFiles.ExpectedFiles) == 0 {
		c.ReadFiles.ExpectedFiles = []string{
			"Kostst_*.csv",
			"Liefer_*.csv",
			"Warengruppe_*.csv",
			"Vpckeinh_*.csv",
			"Artikel_*.csv",
			"Lieferschein_*.csv",
			"Lieferpos_*.csv",
			"Inventur_*.csv",
			"Invposart_*.csv",
			"His_verbrauch_*.csv",
			"Dailytotals_*.csv",
		}
	}
}
