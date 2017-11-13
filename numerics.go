package main

import (
	"log"
	"strings"

	irc "github.com/thoj/go-ircevent"
)

func (b *Bot) addNumerics() {
	b.Connection.AddCallback("001", func(e *irc.Event) {
		if b.Config.Modes != "" {
			b.Connection.SendRawf("MODE %s %s", b.Config.Nick, b.Config.Modes)
		}

		if b.Config.Autojoin != "" {
			b.Connection.Join(b.Config.Autojoin)
		}
	})

	b.Connection.AddCallback("352", func(e *irc.Event) {
		// channel, ident, host, nick, modes, hop, realname := strings.Split(" ", e.Arguments[1:len(e.Arguments)-1])
		s := strings.SplitN(e.Raw, " ", 9)
		log.Printf("%#v\n", s)
	})

	b.Connection.AddCallback("JOIN", func(e *irc.Event) {
		if e.Nick == b.Connection.GetNick() {
			b.Connection.SendRawf("WHO %s", e.Message())
		}

		if e.Nick != b.Connection.GetNick() {
			log.Println(e.Arguments, e.Host, e.Message(), e.Nick, e.Source, e.User)
		}
	})
}
