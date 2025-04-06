package diag

import (
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
)

// CmdDiag is the diagnostic tool.
var CmdDiag = &base.Command{
	Run:       nil,
	Wizard:    nil,
	UsageLine: "slackdump tools",
	Short:     "diagnostic tools",
	Long: `
# Diagnostic tools

Tools command contains different tools, running which may be requested if you open an issue on Github.
`,
	CustomFlags: false,
	FlagMask:    0,
	PrintFlags:  false,
	RequireAuth: false,
	Commands: []*base.Command{
		// cmdEdge,
		cmdEncrypt,
		cmdEzTest,
		cmdInfo,
		cmdObfuscate,
		// cmdRawOutput,
		cmdUninstall,
		cmdRecord,
		// cmdSearch,
		cmdThread,
		cmdHydrate,
		cmdRedownload,
		// cmdWizDebug,
		cmdUnzip,
		cmdConvertV1,
	},
}
