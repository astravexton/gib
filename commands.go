package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"bitbucket.org/nathan93b/gib/ircstructs"

	"golang.org/x/tools/imports"

	humanize "github.com/dustin/go-humanize"
	"github.com/shkh/lastfm-go/lastfm"
	irc "github.com/thoj/go-ircevent"
)

type Tweet struct {
	URL        string `json":url"`
	AuthorName string `json":author_name"`
	AuthorURL  string `json":author_url"`
	HTML       string `json":html"`
}

func stripColors(s string) string {
	r := regexp.MustCompile(`\x03(\d{1,2}(,\d{1,2})?)?`)
	return r.ReplaceAllString(s, "")
}

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

func youtube(query string, v string, key string) string {
	hc := http.Client{}
	if query != "" {
		req, err := http.NewRequest("GET", "https://www.googleapis.com/youtube/v3/search", nil)

		q := req.URL.Query()
		q.Add("q", query)
		q.Add("part", "id")
		q.Add("maxResults", "1")
		q.Add("type", "video")
		q.Add("key", key)
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
		v = ytjson.Items[0].ID.VideoID
	}

	req, err := http.NewRequest("GET", "https://www.googleapis.com/youtube/v3/videos", nil)
	q := req.URL.Query()
	q.Add("part", "id,snippet,contentDetails,statistics,status,liveStreamingDetails")
	q.Add("id", v)
	q.Add("key", key)
	req.URL.RawQuery = q.Encode()

	resp, err := hc.Do(req)
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

	o := fmt.Sprintf("%s | %s | %s views", ytvid.Items[0].Snippet.Title,
		ytvid.Items[0].Snippet.ChannelTitle, humanize.Comma(i))

	if query != "" {
		o = fmt.Sprintf("https://youtu.be/%s | %s", ytvid.Items[0].ID, o)
	}

	return o
}

func getNowPlaying(user string, b *Bot) (lastfm.UserGetRecentTracks, error) {
	return b.LastFMAPI.User.GetRecentTracks(lastfm.P{"user": user})
}

func getTweet(tweet string) string {
	hc := http.Client{}
	u, _ := url.Parse("https://publish.twitter.com/oembed?url=")
	q := u.Query()
	q.Set("url", tweet)
	u.RawQuery = q.Encode()
	req, reqerr := http.NewRequest("GET", u.String(), nil)
	if reqerr != nil {
		return reqerr.Error()
	}
	resp, doerr := hc.Do(req)
	if doerr != nil {
		return doerr.Error()
	}
	defer resp.Body.Close()

	tweetjson := &Tweet{}
	err := json.NewDecoder(resp.Body).Decode(&tweetjson)
	if err != nil {
		return err.Error()
	}
	re := regexp.MustCompile("<[^>]*>")
	stripSpaces := regexp.MustCompile(`\s+`)
	return strings.Trim(html.UnescapeString(stripSpaces.ReplaceAllString(re.ReplaceAllString(tweetjson.HTML, " "), " ")), " ")
}

func (b *Bot) addPrivmsg() {

	gotemplate := `package main
	
	func main() {
		%s	
	}`

	b.Connection.AddCallback("PRIVMSG", func(e *irc.Event) {
		target := e.Arguments[0]

		// if percentCaps(e.Message()) >= 80 && len(e.Message()) >= 10 {
		// 	bot.Kick(e.Nick, target, "Goodbye")
		// 	return
		// }

		tweetregex := regexp.MustCompile(`(https?):\/\/(twitter.com)\/(.*)\/status\/(\d+)`)
		tweet := tweetregex.FindString(e.Message())
		if tweet != "" {
			b.Connection.Privmsg(target, getTweet(tweet))
		}

		if b.Config.Name == "operanet" {

			yt := regexp.MustCompile(`http(?:s?):\/\/(?:www\.)?youtu(?:be\.com\/watch\?v=|\.be\/)([\w\-\_]*)(&(amp;)?‌​[\w\?‌​=]*)?`)
			ytlink := yt.FindStringSubmatch(e.Message())
			if len(ytlink) == 4 && ytlink[1] != "" {
				b.Connection.Privmsg(target, youtube("", ytlink[1], b.Config.APIKeys.Youtube))
			}
		}

		if strings.Split(e.Message(), " ")[0] == ".choose" { //&& b.Config.Name == "subluminal" {
			b.ChooseResult = make(map[string]int)
			s := regexp.MustCompile(`,\s*`)
			choices := s.Split(strings.SplitAfterN(e.Message(), " ", 2)[1], -1)
			for _, c := range choices {
				b.ChooseResult[c] = 0
			}
			r := choices[rand.Intn(len(choices))]
			b.ChooseResult[r]++
			b.Connection.Privmsgf(target, "%s: %s", e.Nick, r)
			go func() {
				time.Sleep(time.Second * 3)
				var o string
				for k, v := range b.ChooseResult {
					o = fmt.Sprintf("%s %s (%d)", o, k, v)
				}
				b.Connection.Privmsgf(target, "%s:%s", e.Nick, o)
				b.ChooseResult = make(map[string]int)
			}()
			return
		}

		for k := range b.ChooseResult {
			// if regexp.MustCompile(fmt.Sprintf(".*: %s", k)).MatchString(regexp.MustCompile(`[^:\w\s]`).ReplaceAllString(e.Message(), "")) == true {
			// if regexp.MustCompile(fmt.Sprintf(".*: %s", k)).MatchString(e.Message()) == true {
			if strings.HasSuffix(e.Message(), k) == true {
				b.ChooseResult[k]++
			}
		}

		trigger := fmt.Sprintf(`^(%s)(.*?)(?:$|\s+)(.*)`, b.Config.Prefix)
		re := regexp.MustCompile(trigger).FindString(e.Message())
		if re != "" {
			s := strings.SplitN(re[1:], " ", 2)
			cmd := stripColors(s[0])
			args := ""
			// argsStripped := ""
			if len(s) > 1 {
				args = s[1]
				// argsStripped = stripColors(s[1])
			}

			switch cmd {
			// case "yt", "youtube", "vid":
			// 	if args != "" {
			// 		b.Connection.Privmsg(target, youtube(args, "", b.Config.APIKeys.Youtube))
			// 	}
			// 	return
			case "quoteadd", "addquote":
				if args == "" {
					b.Connection.Privmsgf(target, "%s: quoteadd <quote>", e.Nick)
					return
				}

				tx, err := b.QuoteDB.Begin()
				if err != nil {
					b.Connection.Privmsgf(target, "error in Begin: %s", err.Error())
					return
				}

				s, err := tx.Prepare("insert into quotes (chan, quote, addedby, time) VALUES (?, ?, ?, ?);")
				if err != nil {
					b.Connection.Privmsgf(target, "error in Prepare: %s", err.Error())
					return
				}
				r, err := s.Exec(target, args, e.Nick, time.Now().Unix())
				if err != nil {
					b.Connection.Privmsgf(target, "error in Exec: %s", err.Error())
					return
				}
				tx.Commit()
				lr, _ := r.LastInsertId()
				b.Connection.Privmsgf(target, "%s: quote #%d added.", e.Nick, lr)
				return

			case "getquote", "quote":
				if args == "" {
					b.Connection.Privmsgf(target, "%s: quote <id>", e.Nick)
					return
				}
				stmt, err := b.QuoteDB.Prepare("select quote, addedby, time from quotes where id = ?")
				if err != nil {
					b.Connection.Privmsgf(target, "error in Prepare: %s", err.Error())
					return
				}
				defer stmt.Close()

				var addedby string
				var quote string
				var time time.Time
				err = stmt.QueryRow(args).Scan(&quote, &addedby, &time)
				if err != nil {
					b.Connection.Privmsgf(target, "No such quote id")
					return
				}
				b.Connection.Privmsgf(target, "Quote #%s: \"%s\" added by %s", args, quote, addedby)
				return

			// case "settime":
			// 	_, err := time.LoadLocation(args)
			// 	if err != nil {
			// 		b.Connection.Privmsgf(target, "%s: %q is not a valid location", e.Nick, args)
			// 		return
			// 	}

			// 	t := ircstructs.TimeData{Name: e.Nick, Timezone: args}
			// 	if err := db.Write("timezones", e.Nick, t); err != nil {
			// 		bot.Connection.Privmsgf(target, "%s: error in adding timezone: %s", e.Nick, err.Error())
			// 		return
			// 	}
			// 	bot.Connection.Privmsgf(target, "%s: done.", e.Nick)

			// case "time":
			// 	if args == "" {
			// 		bot.Connection.Privmsgf(target, "%s: time <nick>", e.Nick)
			// 		return
			// 	}

			// 	t := ircstructs.TimeData{}
			// 	if err := db.Read("timezones", args, &t); err != nil {
			// 		bot.Connection.Privmsgf(target, "%s: %s has not set their timezone.", e.Nick, args)
			// 		return
			// 	}

			// 	loc, _ := time.LoadLocation(t.Timezone)
			// 	cTime := time.Now().In(loc).Format("Mon, 2 Jan 2006 15:04:05 -0700 (MST)")
			// 	bot.Connection.Privmsgf(target, "%s: %s's time: %s", e.Nick, t.Name, cTime)

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
				user := ""
				if args != "" {
					u := ircstructs.LastFM{Name: args}
					if err := b.LastFMDB.Write("lastfm", e.Nick, u); err != nil {
						b.Connection.Privmsgf(target, err.Error())
					}
					user = args
				} else {
					u := ircstructs.LastFM{}
					if err := b.LastFMDB.Read("lastfm", e.Nick, &u); err != nil {
						b.Connection.Privmsgf(target, "%s: to set your last.fm type `np <username>`", e.Nick)
						return
					}
					user = u.Name
				}

				np, err := getNowPlaying(user, b)
				if err != nil {
					b.Connection.Privmsgf(target, "Unable to get now playing info: API may be down")
					return
				}

				if np.Tracks[0].NowPlaying == "true" {
					b.Connection.Privmsgf(target, "%s is listening to: %s - %s", np.User, np.Tracks[0].Artist.Name, np.Tracks[0].Name)
				} else {
					b.Connection.Privmsgf(target, "%s was listening to: %s - %s", np.User, np.Tracks[0].Artist.Name, np.Tracks[0].Name)
				}
				return

			case "ping":
				b.Connection.Privmsg(target, "Pong!")

			case "source":
				b.Connection.Privmsgf(target, "%s: https://bitbucket.org/nathan93b/Gib", e.Nick)

			case "go":
				if stringInSlice(e.Source, b.Config.Admins) == true {
					stub := fmt.Sprintf(gotemplate, args)
					tempfile := filepath.Join(os.TempDir(), "stub.go")

					formatted, err := imports.Process("prog.go", []byte(stub), nil)
					if err != nil {
						b.Connection.Privmsg(target, err.Error())
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
					for _, line := range strings.Split(string(out), "\n") {
						if line != "" {
							b.Connection.Privmsg(target, line)
						}
					}
					return
				}
			}
		}
	})
}

func (b *Bot) addOthers() {
	b.Connection.AddCallback("ERROR", func(e *irc.Event) {
		b.Connection.Disconnect()
		fmt.Println(e.Message())
		os.Exit(1)
	})
}
