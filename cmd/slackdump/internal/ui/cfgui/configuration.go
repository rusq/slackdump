package cfgui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
)

type configuration []group

type group struct {
	name   string
	params []parameter
}

type parameter struct {
	Name        string
	Value       string
	Description string
	Model       tea.Model
}

func effectiveConfig() configuration {
	return configuration{
		{
			name: "Authentication",
			params: []parameter{
				{
					Name:        "Slack Workspace",
					Value:       bootstrap.CurrentWsp(),
					Description: "Currently selected Slack Workspace",
				},
			},
		},
		{
			name: "Timeframe",
			params: []parameter{
				{
					Name:        "Start date",
					Value:       cfg.Oldest.String(),
					Description: "The oldest message to fetch",
				},
				{
					Name:        "End date",
					Value:       cfg.Latest.String(),
					Description: "The newest message to fetch",
				},
			},
		},
		{
			name: "Options",
			params: []parameter{
				{
					Name:        "Enterprise mode",
					Value:       checkbox(cfg.ForceEnterprise),
					Description: "Force enterprise mode",
					Model:       boolUpdateModel{&cfg.ForceEnterprise},
				},
				{
					Name:        "Download files",
					Value:       checkbox(cfg.DownloadFiles),
					Description: "Download files",
					Model:       boolUpdateModel{&cfg.DownloadFiles},
				},
				{
					Name:        "No Chunk Cache",
					Value:       checkbox(cfg.NoChunkCache),
					Description: "Disable chunk cache",
				},
			},
		},
		{
			name: "Various",
			params: []parameter{
				{
					Name:        "API limits file",
					Value:       cfg.ConfigFile,
					Description: "API limits file",
				},
				{
					Name:        "Output",
					Value:       cfg.Output,
					Description: "Output directory",
				},
			},
		},
		{
			name: "Cache",
			params: []parameter{
				{
					Name:        "Local Cache Directory",
					Value:       cfg.LocalCacheDir,
					Description: "Local Cache Directory for user data",
				},
				{
					Name:        "User Cache Retention",
					Value:       cfg.UserCacheRetention.String(),
					Description: "For how long user cache is kept, until it is fetched again",
				},
				{
					Name:        "Disable User Cache",
					Value:       checkbox(cfg.NoUserCache),
					Description: "Disable User Cache",
				},
				{
					Name:        "Disable Chunk Cache",
					Value:       checkbox(cfg.NoChunkCache),
					Description: "Disable Chunk Cache",
				},
			},
		},
	}
}
