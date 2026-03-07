# Compiling from Sources

[Back to User Guide](README.md)

## Install with `go install`

If you have Go installed, you can build and install the latest release directly:

```shell
go install github.com/rusq/slackdump/v4/cmd/slackdump@latest
```

## Build from a Repository Checkout

Clone the repository and build:

```shell
git clone https://github.com/rusq/slackdump.git
cd slackdump
go build -o slackdump ./cmd/slackdump
```

Or run without building a binary:

```shell
go run ./cmd/slackdump
```

See `go.mod` for the minimum required Go version.

## Build with Version Info (Release Build)

```shell
make
```

This produces a binary with the correct version string embedded.  Requires
`make` and the Go toolchain.

[Back to User Guide](README.md)
