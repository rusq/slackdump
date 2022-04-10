module github.com/rusq/slackdump/v2

go 1.18

require (
	github.com/MercuryEngineering/CookieMonster v0.0.0-20180304172713-1584578b3403
	github.com/golang/mock v1.6.0
	github.com/joho/godotenv v1.4.0
	github.com/pkg/errors v0.9.1
	github.com/rusq/dlog v1.3.3
	github.com/rusq/osenv/v2 v2.0.1
	github.com/slack-go/slack v0.10.2
	github.com/stretchr/testify v1.7.1
	golang.org/x/time v0.0.0-20220224211638-0e9765cccd65
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace github.com/slack-go/slack => github.com/rusq/slack v0.10.4
