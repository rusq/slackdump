package auth_ui

// LoginType is the login type, that is used to choose the authentication flow,
// for example login headlessly or interactively.
type LoginType int8

const (
	// LInteractive is the SSO login type (Google, Apple, etc).
	LInteractive LoginType = iota
	// LHeadless is the email/password login type.
	LHeadless
	// LUserBrowser is the google auth option
	LUserBrowser
	// LCancel should be returned if the user cancels the login intent.
	LCancel
)
