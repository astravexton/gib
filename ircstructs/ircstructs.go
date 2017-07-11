package ircstructs

// Config holds bot config
type Config struct {
	Server       string
	SSL          bool
	Nick         string
	Gecos        string
	Sasl         bool
	User         string
	SaslUsername string
	SaslPassword string
	Autojoin     string
	Debug        bool
	Prefix       string
}
