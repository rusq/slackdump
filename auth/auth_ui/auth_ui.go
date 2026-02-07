// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
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
	// LMobileSignin allows to sign in using QR Code
	LMobileSignin
	// LCancel should be returned if the user cancels the login intent.
	LCancel
)
