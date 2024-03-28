package base

// StatusCode is the code returned to the OS.
//
//go:generate stringer -type StatusCode -trimprefix S
type StatusCode uint8

// Status codes returned by the main executable.
const (
	SNoError StatusCode = iota
	SGenericError
	SHelpRequested
	SInvalidParameters
	SAuthError
	SInitializationError
	SApplicationError
	SWorkspaceError
	SCacheError
	SUserError
)
