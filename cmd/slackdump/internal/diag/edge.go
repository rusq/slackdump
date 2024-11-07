package diag

import (
	"context"
	"encoding/json"
	"os"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/edge"
	"github.com/rusq/slackdump/v3/logger"
)

var CmdEdge = &base.Command{
	Run:         runEdge,
	UsageLine:   "slack tools edge",
	Short:       "Edge test",
	RequireAuth: true,
	HideWizard:  true,
	Long: `
# Slack Edge API test tool

Edge test attempts to call the Edge API with the provided credentials.
`,
}

var edgeParams = struct {
	channel string
}{}

func init() {
	CmdEdge.Flag.StringVar(&edgeParams.channel, "channel", "CHY5HUESG", "channel to get users from")
}

func runEdge(ctx context.Context, cmd *base.Command, args []string) error {
	lg := logger.FromContext(ctx)

	prov, err := auth.FromContext(ctx)
	if err != nil {
		base.SetExitStatus(base.SAuthError)
		return err
	}

	cl, err := edge.New(ctx, prov)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	defer cl.Close()
	lg.Print("connected")

	lg.Printf("*** Search for Channels test ***")
	channels, err := cl.SearchChannels(ctx, "")
	if err != nil {
		return err
	}
	if err := save("channels.json", channels); err != nil {
		return err
	}

	// lg.Printf("*** IMs test ***")
	// ims, err := cl.IMList(ctx)
	// if err != nil {
	// 	return err
	// }
	// if err := save("ims.json", ims); err != nil {
	// 	return err
	// }

	// lg.Printf("*** Counts ***")
	// counts, err := cl.ClientCounts(ctx)
	// if err != nil {
	// 	return err
	// }
	// if err := save("counts.json", counts); err != nil {
	// 	return err
	// }

	// lg.Print("*** GetConversationsContext ***")
	// gcc, _, err := cl.GetConversationsContext(ctx, nil)
	// if err != nil {
	// 	return err
	// }
	// if err := save("get_conversations_context.json", gcc); err != nil {
	// 	return err
	// }

	// lg.Print("*** GetUsersInConversationContext ***")
	// if len(gcc) > 0 {
	// 	lg.Printf("using: %s", gcc[0].Name)
	// 	guic, _, err := cl.GetUsersInConversationContext(ctx, &slack.GetUsersInConversationParameters{ChannelID: gcc[0].ID})
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if err := save("get_users_in_conversation_context.json", guic); err != nil {
	// 		return err
	// 	}
	// 	if len(guic) > 0 {
	// 		lg.Print("*** GetUsers ***")
	// 		users, err := cl.GetUsers(ctx, guic...)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		if err := save("get_users.json", users); err != nil {
	// 			return err
	// 		}
	// 	}
	// 	lg.Print("*** Conversations Generic Info ***")
	// 	ci, err := cl.ConversationsGenericInfo(ctx, gcc[0].ID)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if err := save("conversations_generic_info.json", ci); err != nil {
	// 		return err
	// 	}
	// }

	lg.Print("OK")
	return nil
}

func save(filename string, r any) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.Encode(r)
	return err
}
