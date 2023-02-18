module github.com/rusq/slackdump/v2

go 1.18

require (
	github.com/AlecAivazis/survey/v2 v2.3.6
	github.com/MercuryEngineering/CookieMonster v0.0.0-20180304172713-1584578b3403
	github.com/denisbrodbeck/machineid v1.0.1
	github.com/fatih/color v1.13.0
	github.com/golang/mock v1.6.0
	github.com/joho/godotenv v1.4.0
	github.com/playwright-community/playwright-go v0.2000.1
	github.com/rusq/dlog v1.3.3
	github.com/rusq/osenv/v2 v2.0.1
	github.com/rusq/secure v0.0.4
	github.com/rusq/tracer v1.0.1
	github.com/schollz/progressbar/v3 v3.8.6
	github.com/slack-go/slack v0.11.0
	github.com/stretchr/testify v1.7.1
	golang.org/x/net v0.7.0
	golang.org/x/time v0.0.0-20220922220347-f3bd1da661af
)

require (
	github.com/danwakefield/fnmatch v0.0.0-20160403171240-cbb64ac3d964 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.3.1 // indirect
	golang.org/x/crypto v0.0.0-20221012134737-56aed061732a // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/term v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/slack-go/slack => github.com/rusq/slack v0.12.0
