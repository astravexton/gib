package ircstructs

// CommandThing holds stuff
type CommandThing struct {
	commandHooks map[string][]func(s string) string
}

// Config holds bot config
type Config struct {
	Name           string   `json:"name"`
	Active         bool     `json:"active,omitempty"`
	Server         string   `json:"server,omitempty"`
	SSL            bool     `json:"ssl,omitempty"`
	Nick           string   `json:"nick,omitempty"`
	Gecos          string   `json:"gecos,omitempty"`
	Sasl           bool     `json:"sasl,omitempty"`
	User           string   `json:"user,omitempty"`
	SaslUsername   string   `json:"sasl_username,omitempty"`
	SaslPassword   string   `json:"sasl_password,omitempty"`
	Autojoin       string   `json:"autojoin,omitempty"`
	Debug          bool     `json:"debug,omitempty"`
	Prefix         string   `json:"prefix,omitempty"`
	ServerPassword string   `json:"server_password,omitempty"`
	Modes          string   `json:"modes,omitempty"`
	Admins         []string `json:"admins,omitempty"`
}

// Servers holds the array of servers
type Servers struct {
	Server []Config `json:"servers"`
}
