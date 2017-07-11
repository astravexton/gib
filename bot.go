package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/thoj/go-ircevent"

	"Gib/ircstructs"
)

func main() {

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter config name (e.g. freenode.json): ")
	cfile, _ := reader.ReadString('\n')
	cfile = strings.Split(cfile, "\r")[0]

	content, err := ioutil.ReadFile(cfile)

	var conf ircstructs.Config
	err = json.Unmarshal(content, &conf)

	if err != nil {
		fmt.Printf("Err %s", err)
	}

	bot := irc.IRC(conf.Nick, conf.User)
	bot.SASLLogin = conf.SaslUsername
	bot.SASLPassword = conf.SaslPassword
	bot.UseSASL = conf.Sasl
	bot.UseTLS = conf.SSL
	if conf.SSL {
		bot.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	bot.Debug = conf.Debug

	bot.AddCallback("001", func(e *irc.Event) {
		bot.Join(conf.Autojoin)
	})

	bot.AddCallback("PRIVMSG", func(e *irc.Event) {
		target := e.Arguments[0]

		trigger := fmt.Sprintf(`^(?:%s)(.*?)(?:$|\s+)(.*)`, conf.Prefix)
		re := regexp.MustCompile(trigger).FindString(e.Message())
		if re != "" {
			s := strings.SplitN(re[1:], " ", 2)
			cmd := s[0]
			args := ""
			if len(s) > 1 {
				args = s[1]
			}

			fmt.Printf("%s@%s> %s %s\n", e.Nick, target, cmd, args)
		}
	})

	cerr := bot.Connect(conf.Server)

	if cerr != nil {
		fmt.Printf("Err: %s", cerr)
		return
	}

	bot.Loop()
}
