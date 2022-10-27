package base

// Some status codes returned by the main executable.
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
