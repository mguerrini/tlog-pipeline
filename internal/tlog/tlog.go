package tlog

// GeneratedFile representa un archivo XML producido por un generador.
// Cada archivo contiene exactamente una <Transaction> identificada por SeqNum.
type GeneratedFile struct {
	SeqNum     string
	XMLContent string
	NumLines   int
}

// GenerateResult devuelve los archivos producidos y metadata agregada.
type GenerateResult struct {
	Files    []GeneratedFile
	Empty    bool
	NumDocs  int
	NumLines int
}
