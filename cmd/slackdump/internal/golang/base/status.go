package base

// Status codes returned by the main executable.
const (
	SNoError = iota
	SGenericError
	SInvalidParameters
	SHelpRequested
	SAuthError
	SApplicationError
	SWorkspaceError
	SCacheError
)
