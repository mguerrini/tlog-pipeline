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

// DocSeqMap mapea el ID fuente de un documento (LFS_ID, INV_ID, VBR_ID, …)
// al SequenceNumber pre-asignado para ese documento dentro del KST actual.
// Se construye antes de iniciar la generación de XMLs para que todos los
// documentos del KST conozcan su secuencia antes de escribir cualquier archivo.
type DocSeqMap map[string]string
