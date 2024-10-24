package wizard

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/apiconfig"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
)

// initFlags initializes flags based on the key-value pairs.
// Example:
//
//	var (
//		enterpriseMode bool
//		downloadFiles  bool
//	)
//
//	flags, err := initFlags(enterpriseMode, "enterprise", downloadFiles, "files")
//	if err != nil {
//		return err
//	}
func initFlags(keyval ...any) ([]string, error) {
	var flags []string
	if len(keyval)%2 != 0 {
		return flags, errors.New("initFlags: odd number of key-value pairs")
	}
	for i := 0; i < len(keyval); i += 2 {
		if keyval[i].(bool) {
			flags = append(flags, keyval[i+1].(string))
		}
	}
	return flags, nil
}

func Config(ctx context.Context) error {
	const timeExample = "2021-12-31T23:59:59"
	var (
		switches string
		dateFrom string = cfg.Oldest.String()
		dateTo   string = cfg.Latest.String()
		output   string = cfg.StripZipExt(cfg.Output)
	)

	flags, err := initFlags(cfg.ForceEnterprise, "enterprise", cfg.DownloadFiles, "files")
	if err != nil {
		return err
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Output directory").
				Description("Must not exist, will be created").
				Validate(validateNotExist).
				Value(&output),
			huh.NewInput().
				Title("Start date").
				DescriptionFunc(func() string {
					switch dateFrom {
					case "":
						return "From the beginning of times"
					default:
						return "From the specified date, i.e. " + timeExample
					}
				}, &dateFrom).
				Validate(validateDateLayout).
				Value(&dateFrom),
			huh.NewInput().
				Title("End date").
				DescriptionFunc(func() string {
					switch dateTo {
					case "":
						return "Until now"
					default:
						return "Until the specified date, i.e. " + timeExample
					}
				}, &dateTo).
				Validate(validateDateLayout).
				Value(&dateTo),
			huh.NewFilePicker().
				Title("API limits configuration file").
				Description("No file means default limits").
				AllowedTypes([]string{".yaml", ".yml"}).
				Validate(validateAPIconfig).
				Value(&cfg.ConfigFile),
			huh.NewMultiSelect[string]().
				Title("Switches").
				Options(
					huh.NewOption("Enterprise mode", "enterprise"),
					huh.NewOption("Include files and attachments", "files"),
				).Value(&flags).
				DescriptionFunc(func() string {
					switch switches {
					case "enterprise":
						return "Enterprise mode is required if you're running Slack Enterprise Grid"
					case "files":
						return "Files will be downloaded along the messages"
					}
					return ""
				}, &switches),
		),
	).WithTheme(cfg.Theme).WithAccessible(cfg.AccessibleMode)

	if err := form.RunWithContext(ctx); err != nil {
		return err
	}
	// TODO: parse dates
	if err := cfg.Oldest.Set(dateFrom); err != nil {
		return err
	}
	if dateTo == "" {
		cfg.Latest = cfg.TimeValue(time.Now())
	} else {
		if err := cfg.Latest.Set(dateTo); err != nil {
			return err
		}
	}
	if time.Time(cfg.Latest).Before(time.Time(cfg.Oldest)) {
		cfg.Latest, cfg.Oldest = cfg.Oldest, cfg.Latest
	}

	cfg.DownloadFiles, cfg.ForceEnterprise = false, false
	for _, f := range flags {
		switch f {
		case "enterprise":
			cfg.ForceEnterprise = true
		case "files":
			cfg.DownloadFiles = true
		}
	}
	cfg.Output = cfg.StripZipExt(output)

	return nil
}

func validateDateLayout(s string) error {
	if s == "" {
		return nil
	}
	var t cfg.TimeValue
	return t.Set(s)
}

func validateAPIconfig(s string) error {
	if s == "" {
		return nil
	}
	if _, err := os.Stat(s); err != nil {
		return err
	}
	if err := apiconfig.CheckFile(s); err != nil {
		return errors.New("not a valid API limits configuration file")
	}
	return nil
}

func validateNotExist(s string) error {
	if s == "" {
		return errors.New("output directory is required")
	}
	if _, err := os.Stat(s); err == nil {
		return errors.New("output directory already exists")
	}
	return nil
}
