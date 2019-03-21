package main

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
	APIKeys        struct {
		WolframAlpha string `json:"wolframalpha,omitempty"`
		Youtube      string `json:"youtube,omitempty"`
		Weather      string `json:"weather,omitempty"`
		LastFM       struct {
			Key    string `json:"key,key"`
			Secret string `json:"secret,omitempty"`
		} `json:"lastfm,omitempty"`
		Twitter struct {
			ConsumerKey    string `json:"consumer_key,omitempty"`
			ConsumerSecret string `json:"consumer_secret,omitempty"`
			AccessKey      string `json:"access_key,omitempty"`
			AccessToken    string `json:"access_token,omitempty"`
		} `json:"twitter,omitempty"`
	} `json:"apikeys,omitempty"`
}

// Servers holds the array of servers
type Servers struct {
	Server []Config `json:"servers"`
}

// TimeData holds user timezone
type TimeData struct {
	Name     string
	Timezone string
}

// LastFM holds lastfm user
type LastFM struct {
	Name string
}

// Quote holds a user quote
type Quote struct {
	Added     int64
	By        string
	Quote     string
	QuoteID   int
	Upvotes   int
	Downvotes int
}

// YT holds youtube id
type YT struct {
	Items []struct {
		ID struct {
			VideoID string `json:"videoId"`
		} `json:"id"`
	} `json:"Items"`
}

// YTVid holds video info
type YTVid struct {
	Items []struct {
		ID      string `json:"id"`
		Snippet struct {
			Title        string `json:"title"`
			ChannelTitle string `json:"channelTitle"`
		} `json:"snippet"`
		Statistics struct {
			ViewCount string `json:"viewCount"`
		} `json:"statistics"`
	} `json:"items"`
	Kind string `json:"kind"`
}
