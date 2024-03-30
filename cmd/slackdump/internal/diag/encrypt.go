package diag

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
)

// pub   rsa4096 2020-03-22 [SC] [expires: 2029-03-21]
//
//	337530DD887FB454C4FC6E7F23B9DBD7FAE54FCD
//
// uid   <16064414+rusq@users.noreply.github.com>
// sub   rsa4096 2020-03-22 [E] [expires: 2029-03-21]
const pubkey = `-----BEGIN PGP PUBLIC KEY BLOCK-----

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
=aWcB
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

// flags
var gArm bool

func init() {
	if err := initRecipient(); err != nil {
		panic(err)
	}
	CmdEncrypt.Flag.BoolVar(&gArm, "a", false, "shorthand for -armor")
	CmdEncrypt.Flag.BoolVar(&gArm, "armor", false, "armor the output")
}

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
	in, out, arm, err := parseArgs(args)
	if err != nil {
		return err
	}
	defer in.Close()
	defer out.Close()

	var w io.Writer = out
	if arm || gArm {
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
	if _, err := io.Copy(cw, in); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	return nil
}

// parseArgs parses arguments and returns the input and output streams.
//  1. if no arguments are given, input is stdin and output is stdout
//  2. if one argument is given, and it is not a "-", the input is a file
//  3. if two arguments are given, the input is a file, if it's not a "-" otherwise stdin,
//     the output is a file, if it's not a "-" otherwise stdout.
//  4. if more than two arguments are given, it's an error
//  5. if output is stdout, arm the output automatically
func parseArgs(args []string) (in io.ReadCloser, out io.WriteCloser, arm bool, err error) {

	switch len(args) {
	case 0:
		in = os.Stdin
		out = os.Stdout
		arm = true
	case 1:
		if args[0] == "-" {
			in = os.Stdin
		} else {
			in, err = os.Open(args[0])
			if err != nil {
				base.SetExitStatus(base.SApplicationError)
				return nil, nil, false, err
			}
		}
		out = os.Stdout
		arm = true
	case 2:
		if args[0] == "-" {
			in = os.Stdin
		} else {
			in, err = os.Open(args[0])
			if err != nil {
				base.SetExitStatus(base.SApplicationError)
				return nil, nil, false, err
			}
		}
		if args[1] == "-" {
			out = os.Stdout
			arm = true
		} else {
			out, err = os.Create(args[1])
			if err != nil {
				in.Close()
				base.SetExitStatus(base.SApplicationError)
				return nil, nil, false, err
			}
		}
	default:
		base.SetExitStatus(base.SInvalidParameters)
		return nil, nil, false, errors.New("invalid number of arguments")
	}
	return in, out, arm, nil
}
