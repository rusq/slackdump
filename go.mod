module github.com/rusq/slackdump/v2

go 1.18

require (
	github.com/MercuryEngineering/CookieMonster v0.0.0-20180304172713-1584578b3403
	github.com/crowdsecurity/machineid v1.0.2
	github.com/fatih/color v1.13.0
	github.com/gdamore/tcell/v2 v2.5.2
	github.com/golang/mock v1.6.0
	github.com/joho/godotenv v1.4.0
	github.com/playwright-community/playwright-go v0.2000.1
	github.com/rivo/tview v0.0.0-20220728094620-c6cff75ed57b
	github.com/rusq/dlog v1.3.3
	github.com/rusq/osenv/v2 v2.0.1
	github.com/rusq/secure v0.0.3
	github.com/rusq/tracer v1.0.1
	github.com/schollz/progressbar/v3 v3.8.6
	github.com/slack-go/slack v0.11.0
	github.com/stretchr/testify v1.7.1
	golang.org/x/time v0.0.0-20220722155302-e5dcc9cfc0b9
)

require (
	github.com/danwakefield/fnmatch v0.0.0-20160403171240-cbb64ac3d964 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.3.1 // indirect
	golang.org/x/crypto v0.0.0-20220722155217-630584e8d5aa // indirect
	golang.org/x/sys v0.0.0-20220730100132-1609e554cd39 // indirect
	golang.org/x/term v0.0.0-20220722155259-a9ba230a4035 // indirect
	golang.org/x/text v0.3.7 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/slack-go/slack => github.com/rusq/slack v0.11.100

replace github.com/panta/machineid => github.com/crowdsecurity/machineid v1.0.2
