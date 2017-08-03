package ircstructs

// Config holds bot config
type Config struct {
	Server         string        `json:"server,omitempty"`
	SSL            bool          `json:"ssl,omitempty"`
	Nick           string        `json:"nick,omitempty"`
	Gecos          string        `json:"gecos,omitempty"`
	Sasl           bool          `json:"sasl,omitempty"`
	User           string        `json:"user,omitempty"`
	SaslUsername   string        `json:"sasl_username,omitempty"`
	SaslPassword   string        `json:"sasl_password,omitempty"`
	Autojoin       string        `json:"autojoin,omitempty"`
	Debug          bool          `json:"debug,omitempty"`
	Prefix         string        `json:"prefix,omitempty"`
	ServerPassword string        `json:"server_password,omitempty"`
	Modes          string        `json:"modes,omitempty"`
	Admins         []interface{} `json:"admins,omitempty"` // if there is a better way to do this, tell me
}
