package main

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/shkh/lastfm-go/lastfm"
	irc "github.com/thoj/go-ircevent"
	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"
)

var unicodeCharmap map[string]string

type Tweet struct {
	URL        string `json:"url"`
	AuthorName string `json:"author_name"`
	AuthorURL  string `json:"author_url"`
	HTML       string `json:"html"`
}

func stripColors(s string) string {
	r := regexp.MustCompile(`\x03(\d{1,2}(,\d{1,2})?)?`)
	return r.ReplaceAllString(s, "")
}

func StripChars(s string, toStrip string) string {
	r := regexp.MustCompile(toStrip)
	return r.ReplaceAllString(s, " ")
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func getNowPlaying(user string, b *Bot) (lastfm.UserGetRecentTracks, error) {
	return b.LastFMAPI.User.GetRecentTracks(lastfm.P{"user": user})
}

func youtubeSearch(s string) *youtube.SearchResult {
	client := &http.Client{
		Transport: &transport.APIKey{Key: "AIzaSyCV4e9424hlZzBW5bx8Lfkm7BRuVub9h30"},
	}

	service, err := youtube.New(client)
	if err != nil {
		log.Println(err)
		return nil
	}

	call := service.Search.List("id,snippet").Q(s).MaxResults(1)
	response, err := call.Do()
	if err != nil {
		log.Println(err)
		return nil
	}

	return response.Items[0]

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

	b.Connection.AddCallback("PRIVMSG", func(e *irc.Event) {
		target := e.Arguments[0]

		tweetregex := regexp.MustCompile(`(https?):\/\/(twitter.com)\/(.*)\/status\/(\d+)`)
		tweet := tweetregex.FindString(e.Message())
		if tweet != "" {
			tweetID, _ := strconv.ParseInt(strings.Split(tweet, "/")[5], 10, 64)
			tweet, _, err := b.TwitterClient.Statuses.Show(tweetID, nil)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			t, _ := tweet.CreatedAtTime()
			b.Connection.Privmsgf(target, "\x0302%s\x03 (@%s) %s", StripChars(strings.Replace(tweet.Text, "\x0A", " ", -1), `\s+`), tweet.User.ScreenName, t.Format("January 2, 2006"))
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
			// go func() {
			// 	time.Sleep(time.Second * 3)
			// 	var o string
			// 	for k, v := range b.ChooseResult {
			// 		o = fmt.Sprintf("%s %s (%d)", o, k, v)
			// 	}
			// 	b.Connection.Privmsgf(target, "%s:%s", e.Nick, o)
			// 	b.ChooseResult = make(map[string]int)
			// }()
			return
		}

		// for k := range b.ChooseResult {
		// 	// if regexp.MustCompile(fmt.Sprintf(".*: %s", k)).MatchString(regexp.MustCompile(`[^:\w\s]`).ReplaceAllString(e.Message(), "")) == true {
		// 	// if regexp.MustCompile(fmt.Sprintf(".*: %s", k)).MatchString(e.Message()) == true {
		// 	if strings.HasSuffix(e.Message(), k) == true {
		// 		b.ChooseResult[k]++
		// 	}
		// }

		trigger := fmt.Sprintf(`^(%s)(.*?)(?:$|\s+)(.*)`, b.Config.Prefix)
		re := regexp.MustCompile(trigger).FindString(e.Message())
		if re != "" {
			s := strings.SplitN(re[1:], " ", 2)
			cmd := stripColors(s[0])
			args := ""
			if len(s) > 1 {
				args = s[1]
			}

			switch cmd {
			case "wa":
				ans, err := b.WolframAlpha.Ask(args)
				if err != nil {
					b.Connection.Privmsg(target, err.Error())
					return
				}
				if ans != "" {
					junk := len(fmt.Sprintf("%s%s%sspyker.zyrio.network", b.Config.Nick, b.Config.User, target))
					maxLen := 496 - junk
					start := 0

					for {
						if len(ans) >= maxLen {
							toSend := ans[start:maxLen]
							ans = ans[maxLen:]
							b.Connection.Privmsg(target, toSend)
						} else {
							b.Connection.Privmsg(target, ans)
							break
						}
					}

					// b.Connection.Privmsg(target, ans)
				}
				return

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

				var addedBy string
				var quote string
				var timeAdded time.Time
				err = stmt.QueryRow(args).Scan(&quote, &addedBy, &timeAdded)
				if err != nil {
					b.Connection.Privmsgf(target, "No such quote id")
					return
				}

				// _, _, day, _, _, _ := diff(timeAdded, time.Now())
				// added := fmt.Sprintf("%d day(s)", day)
				added := humanize.Time(timeAdded)
				b.Connection.Privmsgf(target, "Quote #%s: \"%s\" added by %s %s", args, quote, addedBy, added)
				return

			case "np":
				np := lastfm.UserGetRecentTracks{}
				user := ""
				if args != "" {
					u := LastFM{Name: args}
					if err := b.LastFMDB.Write("lastfm", e.Nick, u); err != nil {
						b.Connection.Privmsgf(target, err.Error())
					}
					user = args
				} else {
					u := LastFM{}
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

				if len(np.Tracks) == 0 {
					b.Connection.Privmsgf(target, "This user hasn't listened to anything yet")
					return
				}

				ytResult := youtubeSearch(fmt.Sprintf("%s - %s", np.Tracks[0].Artist.Name, np.Tracks[0].Name))
				videoURL := ""

				if ytResult.Id.VideoId != "" {
					videoURL = fmt.Sprintf(" (YouTube? - youtu.be/%s)", ytResult.Id.VideoId)
				}

				if np.Tracks[0].NowPlaying == "true" {

					b.Connection.Privmsgf(target, "%s is listening to: %s - %s%s", np.User, np.Tracks[0].Artist.Name, np.Tracks[0].Name, videoURL)
				} else {
					b.Connection.Privmsgf(target, "%s was listening to: %s - %s%s", np.User, np.Tracks[0].Artist.Name, np.Tracks[0].Name, videoURL)
				}

				return

			case "ping":
				b.Connection.Privmsg(target, "Pong!")

			case "eggs":
				b.Connection.Privmsg(target, "Bacon")

			case "coffee":
				b.Connection.Privmsg(target, "abzde: coffee")

			case "source":
				b.Connection.Privmsgf(target, "%s: https://bitbucket.org/nathan93b/Gib", e.Nick)

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
