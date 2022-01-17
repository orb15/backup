package domain

//CommandOpts holds command line options to override default config
type CommandOpts struct {

	//UseDebugLogger should be set true when debug-level logging is needed
	UseDebugLogger bool

	//Dryrun should be set true to test configuration and AWS connectivity
	Dryrun bool

	//Reprocess should be set true to reprocess a JSON file of previously-failed files
	Reprocess bool

	//NoConfirm should be set trueif, during Reprocessing, the confirmation menu should be skipped
	NoConfirm bool
}
