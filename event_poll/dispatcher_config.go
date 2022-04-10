package ddio

type DisPatcherConfig struct {
	ConnEvent EventFlags
	ConnHandler ConnectionEventHandler
	ConnErrHandler ErrorHandler
}
