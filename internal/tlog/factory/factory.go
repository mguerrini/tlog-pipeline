package factory

import (
	"github.com/opessa/tlog-pipeline/internal/tlog"
	"github.com/opessa/tlog-pipeline/internal/tlog/adjustment"
	"github.com/opessa/tlog-pipeline/internal/tlog/cierre"
	"github.com/opessa/tlog-pipeline/internal/tlog/count"
	fiscaldoc_fc "github.com/opessa/tlog-pipeline/internal/tlog/fiscaldoc_fc"
	fiscaldoc_nc "github.com/opessa/tlog-pipeline/internal/tlog/fiscaldoc_nc"
	"github.com/opessa/tlog-pipeline/internal/tlog/reception"
	return_ "github.com/opessa/tlog-pipeline/internal/tlog/return_"
	"github.com/opessa/tlog-pipeline/internal/tlog/transfer"
)

// AllGenerators devuelve la lista de generators en el orden canónico NNNN.
func AllGenerators() []tlog.Generator {
	return []tlog.Generator{
		reception.Generator{},    // 0001
		return_.Generator{},      // 0002
		transfer.Generator{},     // 0003
		adjustment.Generator{},   // 0004
		count.Generator{},        // 0005
		fiscaldoc_fc.Generator{}, // 0006
		fiscaldoc_nc.Generator{}, // 0007
		cierre.Generator{},       // 0008
	}
}
