package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/muesli/cache2go"
	"github.com/thoj/go-ircevent"

	"Gib/ircstructs"
	"os/exec"
)

func stringInSlice(a string, list []interface{}) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func main() {

	type join struct {
		nick    string
		time    time.Time
		channel string
	}

	gotemplate := `package main

import (
	"fmt"
)

func main() {
	%s	
}`

	cache := cache2go.Cache("usercache")
	cache.SetAddedItemCallback(func(e *cache2go.CacheItem) {
		fmt.Println("Adding:", e.Key())
	})
	cache.SetAboutToDeleteItemCallback(func(e *cache2go.CacheItem) {
		fmt.Println("Deleting:", e.Key())
	})

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

	bot.Password = conf.ServerPassword

	bot.AddCallback("ERROR", func(e *irc.Event) {
		fmt.Print("ERROR")
		bot.Disconnect()
	})

	bot.AddCallback("001", func(e *irc.Event) {
		bot.SendRawf("MODE %s %s", conf.Nick, conf.Modes)
		if conf.Autojoin != "" {
			bot.Join(conf.Autojoin)
		}
	})

	bot.AddCallback("JOIN", func(e *irc.Event) {
		if e.Nick == conf.Nick {
			return
		}
		target := e.Arguments[0]
		j := join{e.Nick, time.Now(), target}
		cache.Add(e.Nick, 5*time.Second, &j)
	})

	bot.AddCallback("PRIVMSG", func(e *irc.Event) {
		target := e.Arguments[0]

		if strings.Split(e.Message(), " ")[0] == ".choose" {
			choices := strings.Split(strings.Split(e.Message(), " ")[1], ",")
			bot.Privmsgf(target, "%s: %s", e.Nick, choices[rand.Intn(len(choices))])
		}

		trigger := fmt.Sprintf(`^(?:%s)(.*?)(?:$|\s+)(.*)`, conf.Prefix)
		re := regexp.MustCompile(trigger).FindString(e.Message())
		if re != "" {
			s := strings.SplitN(re[1:], " ", 2)
			cmd := s[0]
			args := ""
			if len(s) > 1 {
				args = s[1]
			}

			// if cmd == "compare" {
			// 	dmp := diffmatchpatch.New()
			// 	diffs := dmp.DiffMain(strings.SplitN(args, " ", 2)[0], strings.SplitN(args, " ", 2)[1], false)
			// 	bot.Privmsg(target, dmp.DiffNormalText(diffs))
			// 	return
			// }

			if cmd == "ping" {
				bot.Privmsg(target, "Pong!")
				return
			}

			if cmd == "source" {
				bot.Privmsgf(target, "%s: https://bitbucket.org/nathan93b/Gib", e.Nick)
				return
			}

			if stringInSlice(e.Source, conf.Admins) == true {
				if cmd == "go" {
					stub := fmt.Sprintf(gotemplate, args)
					err := ioutil.WriteFile("C:/Temp/stub.go", []byte(stub), 0644)
					if err != nil {
						fmt.Print(err)
						return
					}

					r := exec.Command("go.exe", "run", "C://Temp//stub.go")

					out, err := r.Output()
					if err != nil {
						fmt.Print(err)
					}
					bot.Privmsg(target, string(out))
					return
				}
			}
		}

		// leave this till last

		// res, err := cache.Value(e.Nick)
		// if err != nil {
		// 	return
		// }

		// if time.Now().Unix() <= res.Data().(*join).time.Add(3*time.Second).Unix() {
		// 	cache.Delete(e.Nick)
		// 	bot.Kick(e.Nick, target, "Goodbye")
		// }

	})

	cerr := bot.Connect(conf.Server)

	if cerr != nil {
		fmt.Printf("Err: %s", cerr)
		return
	}

	bot.Loop()
}
