module github.com/rusq/slackdump

go 1.17

require (
	github.com/joho/godotenv v1.4.0
	github.com/pkg/errors v0.9.1
	github.com/rusq/dlog v1.3.3
	github.com/slack-go/slack v0.9.5
	github.com/stretchr/testify v1.4.0
	golang.org/x/time v0.0.0-20211116232009-f0f3c7e86c11
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v2 v2.2.2 // indirect
)

replace github.com/slack-go/slack => github.com/rusq/slack v0.9.6
