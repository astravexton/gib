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
	"time"

	"git.zyrio.network/astra/wolframalpha"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	_ "github.com/mattn/go-sqlite3"
	scribble "github.com/nanobox-io/golang-scribble"
	"github.com/shkh/lastfm-go/lastfm"
	"github.com/thoj/go-ircevent"
)

// Bot holds bot stuffs
type Bot struct {
	Connection    *irc.Connection
	Config        Config
	QuoteDB       *sql.DB
	LastFMDB      *scribble.Driver
	LastFMAPI     *lastfm.Api
	ChooseResult  map[string]int
	TwitterClient *twitter.Client
	WolframAlpha  *wolframalpha.WolframProvider
}

func main() {

	configFile := flag.String("config", "config.json", "Path to config file to use")
	flag.Parse()

	var bot Bot

	done := make(chan struct{})

	cfile, err := ioutil.ReadFile(*configFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = json.Unmarshal(cfile, &bot.Config)
	if err != nil {
		fmt.Println(err)
		return
	}

	config := oauth1.NewConfig(bot.Config.APIKeys.Twitter.ConsumerKey, bot.Config.APIKeys.Twitter.ConsumerSecret)
	token := oauth1.NewToken(bot.Config.APIKeys.Twitter.AccessKey, bot.Config.APIKeys.Twitter.AccessToken)
	httpClient := config.Client(oauth1.NoContext, token)
	bot.TwitterClient = twitter.NewClient(httpClient)

	bot.WolframAlpha = wolframalpha.NewWolframProvider()
	bot.WolframAlpha.SetApiKey(bot.Config.APIKeys.WolframAlpha)

	bot.LastFMAPI = lastfm.New(bot.Config.APIKeys.LastFM.Key, bot.Config.APIKeys.LastFM.Secret)
	bot.LastFMDB, _ = scribble.New(fmt.Sprintf("data/%s/", bot.Config.Name), nil)

	bot.QuoteDB, _ = sql.Open("sqlite3", fmt.Sprintf("data/%s/quotes.db", bot.Config.Name))
	defer bot.QuoteDB.Close()

	_, err = bot.QuoteDB.Exec("create table if not exists quotes (id integer not null primary key, chan text, quote text, addedby string, time datetime);")
	if err != nil {
		log.Panic(err)
	}

	bot.Connection = irc.IRC(bot.Config.Nick, bot.Config.User)
	bot.Connection.SASLLogin = bot.Config.SaslUsername
	bot.Connection.SASLPassword = bot.Config.SaslPassword
	bot.Connection.UseSASL = bot.Config.Sasl
	bot.Connection.UseTLS = bot.Config.SSL
	if bot.Config.SSL {
		bot.Connection.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
	bot.Connection.Debug = bot.Config.Debug
	bot.Connection.Password = bot.Config.ServerPassword

	bot.addPrivmsg()
	bot.addNumerics()
	bot.addOthers()

	cerr := bot.Connection.Connect(bot.Config.Server)

	if cerr != nil {
		fmt.Printf("Err: %s", cerr)
		return
	}

	go bot.Connection.Loop()

	done <- struct{}{}
	<-done

}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func diff(a, b time.Time) (year, month, day, hour, min, sec int) {
	if a.Location() != b.Location() {
		b = b.In(a.Location())
	}
	if a.After(b) {
		a, b = b, a
	}
	y1, M1, d1 := a.Date()
	y2, M2, d2 := b.Date()

	h1, m1, s1 := a.Clock()
	h2, m2, s2 := b.Clock()

	year = int(y2 - y1)
	month = int(M2 - M1)
	day = int(d2 - d1)
	hour = int(h2 - h1)
	min = int(m2 - m1)
	sec = int(s2 - s1)

	// Normalize negative values
	if sec < 0 {
		sec += 60
		min--
	}
	if min < 0 {
		min += 60
		hour--
	}
	if hour < 0 {
		hour += 24
		day--
	}
	if day < 0 {
		// days in month:
		t := time.Date(y1, M1, 32, 0, 0, 0, 0, time.UTC)
		day += 32 - t.Day()
		month--
	}
	if month < 0 {
		month += 12
		year--
	}

	return
}
