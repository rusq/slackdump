package app

type Config struct {
	Creds     SlackCreds
	ListFlags ListFlags

	Output Output

	ChannelIDs   []string
	IncludeFiles bool
}

type Output struct {
	Filename string
	Format   string
}

func (out Output) FormatValid() bool {
	return out.Format != "" && (out.Format == OutputTypeJSON ||
		out.Format == OutputTypeText)
}

func (out Output) IsText() bool {
	return out.Format == OutputTypeText
}

type SlackCreds struct {
	Token  string
	Cookie string
}

func (c SlackCreds) Valid() bool {
	return c.Token != "" && c.Cookie != ""
}

type ListFlags struct {
	Users    bool
	Channels bool
}

func (lf ListFlags) FlagsPresent() bool {
	return lf.Users || lf.Channels
}
