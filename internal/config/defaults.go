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
	// Si no se configuró ningún log (sección omitida en el JSON), habilitarlos
	// todos para mantener compatibilidad con configs previas.
	l := c.Logs
	if !l.PipelineEnabled && !l.DayStatusEnabled && !l.SQLDBLoad && !l.OrphansReport {
		c.Logs = Logs{
			PipelineEnabled:  true,
			DayStatusEnabled: true,
			SQLDBLoad:        true,
			OrphansReport:    true,
		}
	}
	// Si no se configuró ningún output (sección omitida en el JSON), generar
	// los 8 TLOGs para mantener compatibilidad con configs previas.
	o := c.Output
	if !o.Cierre && !o.InventoryReception && !o.InventoryFiscalDocFC &&
		!o.InventoryFiscalDocNC && !o.InventoryReturn && !o.InventoryAdjustment &&
		!o.InventoryCount && !o.InventoryTransfer {
		c.Output = Output{
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
