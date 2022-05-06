module github.com/rusq/slackdump/v2

go 1.18

require (
	github.com/MercuryEngineering/CookieMonster v0.0.0-20180304172713-1584578b3403
	github.com/fatih/color v1.13.0
	github.com/golang/mock v1.6.0
	github.com/joho/godotenv v1.4.0
	github.com/pkg/errors v0.9.1
	github.com/playwright-community/playwright-go v0.2000.1
	github.com/rusq/dlog v1.3.3
	github.com/rusq/osenv/v2 v2.0.1
	github.com/rusq/tracer v1.0.0
	github.com/slack-go/slack v0.10.2
	github.com/stretchr/testify v1.7.1
	golang.org/x/time v0.0.0-20220224211638-0e9765cccd65
)

require (
	github.com/danwakefield/fnmatch v0.0.0-20160403171240-cbb64ac3d964 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/mattn/go-colorable v0.1.9 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace github.com/slack-go/slack => github.com/rusq/slack v0.10.4
