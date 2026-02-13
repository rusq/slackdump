#!/bin/zsh

OLD="github.com/rusq/slackdump/v3"
NEW="github.com/rusq/slackdump/v4"

find . -type f -name "*.go" -print0 | while IFS= read -r -d '' file; do
  sed -i '' "s|$OLD|$NEW|g" "$file"
done

# Format all Go files
go fmt ./...
