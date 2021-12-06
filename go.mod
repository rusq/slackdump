module github.com/rusq/slackdump

go 1.13

require (
	github.com/joho/godotenv v1.4.0
	github.com/pkg/errors v0.9.1
	github.com/slack-go/slack v0.9.5
	github.com/stretchr/testify v1.4.0 // indirect
	golang.org/x/time v0.0.0-20211116232009-f0f3c7e86c11
)

replace github.com/slack-go/slack => github.com/rusq/slack v0.9.6
