package diag

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/rusq/dlog"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
)

// pub   rsa4096 2020-03-22 [SC] [expires: 2024-03-22]
// 337530DD887FB454C4FC6E7F23B9DBD7FAE54FCD
// pubkey: <16064414+rusq@users.noreply.github.com>
// sub   rsa4096 2020-03-22 [E] [expires: 2024-03-22]
const pubkey = `-----BEGIN PGP PUBLIC KEY BLOCK-----

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
=QFMD
-----END PGP PUBLIC KEY BLOCK-----
`

var CmdEncrypt = &base.Command{
	Run:       runEncrypt,
	UsageLine: "slackdump tools encrypt [flags] file",
	Short:     "encrypts a file to post in github issues",
	Long: `
# Command Encrypt

Encrypt a file with the developer key to attach to a github issue or send
as a message.

It uses the assymetric encryption (GPG) to encrypt the file with the
developer key, and can only be decrypted by the developer.

## Usage

Encrypt a file to attach as a file to github issue:

	$ slackdump tools encrypt -a file

Encrypt a file to post as a message (for small files):

	$ slackdump tools encrypt -a file

`,
	FlagMask:   cfg.OmitAll,
	PrintFlags: true,
}

var recipient *openpgp.Entity

func init() {
	if err := initRecipient(); err != nil {
		panic(err)
	}
	CmdEncrypt.Flag.BoolVar(&arm, "a", false, "shorthand for -armor")
	CmdEncrypt.Flag.BoolVar(&arm, "armor", false, "armor the output")
}

var arm bool

func initRecipient() error {
	block, err := armor.Decode(strings.NewReader(pubkey))
	if err != nil {
		base.SetExitStatus(base.SInvalidParameters)
		return err
	}
	if block.Type != openpgp.PublicKeyType {
		return errors.New("invalid public key")
	}
	reader := packet.NewReader(block.Body)
	recipient, err = openpgp.ReadEntity(reader)
	if err != nil {
		return err
	}
	return nil
}

func runEncrypt(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("must specify a file to encrypt")
	}

	f, err := os.Open(args[0])
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	defer f.Close()

	outfile := args[0] + ".gpg"
	if arm {
		outfile += ".asc"
	}
	if len(args) == 2 {
		outfile = args[1]
	}

	out, err := os.Create(outfile)
	if err != nil {
		return err
	}
	defer out.Close()

	var w io.Writer = out
	if arm {
		// arm if requested
		aw, err := armor.Encode(out, "PGP MESSAGE", nil)
		if err != nil {
			base.SetExitStatus(base.SApplicationError)
			return err
		}
		defer aw.Close()
		w = aw
	}

	cw, err := openpgp.Encrypt(w, []*openpgp.Entity{recipient}, nil, &openpgp.FileHints{IsBinary: true}, nil)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	defer cw.Close()
	if _, err := io.Copy(cw, f); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	dlog.Printf("encrypted %s to %s", args[0], out.Name())
	return nil
}
