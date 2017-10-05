package main

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/dustin/go-humanize"

	"golang.org/x/tools/imports"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nanobox-io/golang-scribble"
	"github.com/shkh/lastfm-go/lastfm"
	"github.com/thoj/go-ircevent"

	"bitbucket.org/nathan93b/gib/ircstructs"
)

func percentCaps(s string) int {
	var u float64

	re := regexp.MustCompile(`\W`)
	s = re.ReplaceAllString(s, "")

	for _, a := range s {
		if unicode.IsUpper(a) {
			u = u + 1
		}
	}

	return int(u / float64(len(s)) * 100)
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func youtube(query string) string {
	hc := http.Client{}
	req, err := http.NewRequest("GET", "https://www.googleapis.com/youtube/v3/search", nil)

	q := req.URL.Query()
	q.Add("q", query)
	q.Add("part", "id")
	q.Add("maxResults", "1")
	q.Add("type", "video")
	q.Add("key", "AIzaSyBn1mBgMwk25d-sFmcdlHI61TJTyR_nado")
	req.URL.RawQuery = q.Encode()

	resp, err := hc.Do(req)
	if err != nil {
		return err.Error()
	}

	defer resp.Body.Close()

	ytjson := &ircstructs.YT{}
	err = json.NewDecoder(resp.Body).Decode(&ytjson)
	if err != nil {
		return err.Error()
	}

	if len(ytjson.Items) == 0 {
		return "No videos found"
	}

	req, err = http.NewRequest("GET", "https://www.googleapis.com/youtube/v3/videos", nil)
	q = req.URL.Query()
	q.Add("part", "id,snippet,contentDetails,statistics,status,liveStreamingDetails")
	q.Add("id", ytjson.Items[0].ID.VideoID)
	q.Add("key", "AIzaSyBn1mBgMwk25d-sFmcdlHI61TJTyR_nado")
	req.URL.RawQuery = q.Encode()

	resp, err = hc.Do(req)
	if err != nil {
		return err.Error()
	}

	defer resp.Body.Close()

	ytvid := &ircstructs.YTVid{}
	err = json.NewDecoder(resp.Body).Decode(&ytvid)
	if err != nil {
		return err.Error()
	}

	i, _ := strconv.ParseInt(ytvid.Items[0].Statistics.ViewCount, 10, 64)

	if len(ytvid.Items) == 0 {
		return "No videos found"
	}

	return fmt.Sprintf("https://youtu.be/%s | %s | %s | %s views",
		ytvid.Items[0].ID, ytvid.Items[0].Snippet.Title,
		ytvid.Items[0].Snippet.ChannelTitle,
		humanize.Comma(i))
}

func main() {

	configFile := flag.String("config", "config.json", "Path to config file to use")
	flag.Parse()

	lastfmapi := lastfm.New("ca347092b54af2ac4cb1f71034c3dd67", "c37e37b32dbe58e2a8cd9959844b8bb6")

	gotemplate := `package main

func main() {
	%s	
}`

	done := make(chan struct{})

	cfile, err := ioutil.ReadFile(*configFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	var conf ircstructs.Config
	err = json.Unmarshal(cfile, &conf)
	if err != nil {
		fmt.Println(err)
		return
	}

	db, err := scribble.New(fmt.Sprintf("data/%s/", conf.Name), nil)
	if err != nil {
		log.Panic(err)
	}

	quotedb, err := sql.Open("sqlite3", fmt.Sprintf("data/%s/quotes.db", conf.Name))
	if err != nil {
		log.Panic(err)
	}
	defer quotedb.Close()

	appingdb, err := sql.Open("sqlite3", fmt.Sprintf("data/%s/apping.db", conf.Name))
	if err != nil {
		log.Panic(err)
	}
	defer appingdb.Close()

	sqlStmt := `create table if not exists fappers (id integer not null primary key, nick text);
	create table if not exists faps (id integer not null primary key, fapperid integer, starttime timedaye, endtime timedate);`
	_, err = appingdb.Exec(sqlStmt)
	if err != nil {
		log.Panic(err)
	}

	_, err = quotedb.Exec("create table if not exists quotes (id integer not null primary key, chan text, quote text, addedby string, time datetime);")
	if err != nil {
		log.Panic(err)
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
		bot.Disconnect()
		fmt.Println(e.Message())
		os.Exit(1)
	})

	bot.AddCallback("001", func(e *irc.Event) {
		if conf.Modes != "" {
			bot.SendRawf("MODE %s %s", conf.Nick, conf.Modes)
		}

		if conf.Autojoin != "" {
			bot.Join(conf.Autojoin)
		}
	})

	bot.AddCallback("PRIVMSG", func(e *irc.Event) {
		target := e.Arguments[0]

		// if percentCaps(e.Message()) >= 80 && len(e.Message()) >= 10 {
		// 	bot.Kick(e.Nick, target, "Goodbye")
		// 	return
		// }

		if strings.Split(e.Message(), " ")[0] == ".choose" && conf.Name == "subluminal" {
			s := regexp.MustCompile(`,\s*`)
			choices := s.Split(strings.SplitAfterN(e.Message(), " ", 2)[1], -1)
			bot.Privmsgf(target, "%s: %s", e.Nick, choices[rand.Intn(len(choices))])
			return
		}

		trigger := fmt.Sprintf(`^(%s)(.*?)(?:$|\s+)(.*)`, conf.Prefix)
		re := regexp.MustCompile(trigger).FindString(e.Message())
		if re != "" {
			s := strings.SplitN(re[1:], " ", 2)
			cmd := s[0]
			args := ""
			if len(s) > 1 {
				args = s[1]
			}

			switch cmd {
			case "yt", "youtube", "vid":
				bot.Privmsg(target, youtube(args))
				return
			case "quoteadd", "addquote":
				if args == "" {
					bot.Privmsgf(target, "%s: quoteadd <quote>", e.Nick)
					return
				}

				tx, err := quotedb.Begin()
				if err != nil {
					bot.Privmsgf(target, "error in Begin: %s", err.Error())
					return
				}

				s, err := tx.Prepare("insert into quotes (chan, quote, addedby, time) VALUES (?, ?, ?, ?);")
				if err != nil {
					bot.Privmsgf(target, "error in Prepare: %s", err.Error())
					return
				}
				r, err := s.Exec(target, args, e.Nick, time.Now().Unix())
				if err != nil {
					bot.Privmsgf(target, "error in Exec: %s", err.Error())
					return
				}
				tx.Commit()
				lr, _ := r.LastInsertId()
				bot.Privmsgf(target, "%s: quote #%d added.", e.Nick, lr)
				return

			case "getquote", "quote":
				if args == "" {
					bot.Privmsgf(target, "%s: quote <id>", e.Nick)
					return
				}
				stmt, err := quotedb.Prepare("select quote, addedby, time from quotes where id = ?")
				if err != nil {
					bot.Privmsgf(target, "error in Prepare: %s", err.Error())
					return
				}
				defer stmt.Close()

				var addedby string
				var quote string
				var time time.Time
				err = stmt.QueryRow(args).Scan(&quote, &addedby, &time)
				if err != nil {
					bot.Privmsgf(target, "No such quote id")
					return
				}
				bot.Privmsgf(target, "Quote #%s: \"%s\" added by %s", args, quote, addedby)
				return

			case "settime":
				_, err := time.LoadLocation(args)
				if err != nil {
					bot.Privmsgf(target, "%s: %q is not a valid location", e.Nick, args)
					return
				}

				t := ircstructs.TimeData{Name: e.Nick, Timezone: args}
				if err := db.Write("timezones", e.Nick, t); err != nil {
					bot.Privmsgf(target, "%s: error in adding timezone: %s", e.Nick, err.Error())
					return
				}
				bot.Privmsgf(target, "%s: done.", e.Nick)

			case "time":
				if args == "" {
					bot.Privmsgf(target, "%s: time <nick>", e.Nick)
					return
				}

				t := ircstructs.TimeData{}
				if err := db.Read("timezones", args, &t); err != nil {
					bot.Privmsgf(target, "%s: %s has not set their timezone.", e.Nick, args)
					return
				}

				loc, _ := time.LoadLocation(t.Timezone)
				cTime := time.Now().In(loc).Format("Mon, 2 Jan 2006 15:04:05 -0700 (MST)")
				bot.Privmsgf(target, "%s: %s's time: %s", e.Nick, t.Name, cTime)

			// case "apping":
			// 	tx, err := appingdb.Begin()
			// 	if err != nil {
			// 		bot.Privmsg(target, err.Error())
			// 		return
			// 	}

			// 	stmt, _ := tx.Prepare("select count(*) from fappers where nick = ?")
			// 	defer stmt.Close()
			// 	var count int
			// 	err = stmt.QueryRow(e.Nick).Scan(&count)
			// 	if err != nil {
			// 		bot.Privmsg(target, err.Error())
			// 		return
			// 	}

			// 	if count == 0 {

			// 	}

			// case "apped":
			// 	fmt.Println("apped")

			// case "appstats":
			// 	fmt.Println("appstats")

			case "np":
				np := lastfm.UserGetRecentTracks{}
				if args != "" {
					u := ircstructs.LastFM{Name: args}
					if err := db.Write("lastfm", e.Nick, u); err != nil {
						bot.Privmsgf(target, err.Error())
					}
					np, err = lastfmapi.User.GetRecentTracks(lastfm.P{"user": args})
				} else {
					u := ircstructs.LastFM{}
					if err := db.Read("lastfm", e.Nick, &u); err != nil {
						bot.Privmsgf(target, "%s: to set your last.fm type `np <username>`", e.Nick)
						return
					}
					np, err = lastfmapi.User.GetRecentTracks(lastfm.P{"user": u.Name})
				}
				if err != nil {
					log.Print(err)
				}
				if np.Tracks[0].NowPlaying == "true" {
					bot.Privmsgf(target, "%s is listening to: %s - %s", np.User, np.Tracks[0].Artist.Name, np.Tracks[0].Name)
				} else {
					bot.Privmsgf(target, "%s was listening to: %s - %s", np.User, np.Tracks[0].Artist.Name, np.Tracks[0].Name)
				}
				return

			case "ping":
				bot.Privmsg(target, "Pong!")

			case "source":
				bot.Privmsgf(target, "%s: https://bitbucket.org/nathan93b/Gib", e.Nick)

			case "go":
				if stringInSlice(e.Source, conf.Admins) == true {
					stub := fmt.Sprintf(gotemplate, args)
					tempfile := filepath.Join(os.TempDir(), "stub.go")

					formatted, err := imports.Process("prog.go", []byte(stub), nil)
					if err != nil {
						bot.Privmsg(target, err.Error())
						return
					}

					err = ioutil.WriteFile(tempfile, []byte(formatted), 0644)
					if err != nil {
						fmt.Print(err)
						return
					}

					r := exec.Command("go", "run", tempfile)
					out, err := r.Output()
					if err != nil {
						fmt.Println("Error in running:", err)
						return
					}
					fmt.Println(string(out))
					for _, line := range strings.Split(string(out), "\n") {
						bot.Privmsg(target, line)
					}
					return
				}
			}
		}

	})

	cerr := bot.Connect(conf.Server)

	if cerr != nil {
		fmt.Printf("Err: %s", cerr)
		return
	}

	go bot.Loop()

	done <- struct{}{}
	<-done

}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}
