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

	scribble "github.com/nanobox-io/golang-scribble"
	"github.com/shkh/lastfm-go/lastfm"

	_ "github.com/mattn/go-sqlite3"
	"github.com/thoj/go-ircevent"

	"bitbucket.org/nathan93b/gib/ircstructs"
)

type Bot struct {
	Connection   *irc.Connection
	Config       ircstructs.Config
	QuoteDB      *sql.DB
	LastFMDB     *scribble.Driver
	LastFMAPI    *lastfm.Api
	ChooseResult map[string]int
}

func main() {

	var bot Bot

	configFile := flag.String("config", "config.json", "Path to config file to use")
	flag.Parse()

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

	bot.LastFMAPI = lastfm.New(bot.Config.APIKeys.LastFM.Key, bot.Config.APIKeys.LastFM.Secret)

	bot.LastFMDB, _ = scribble.New(fmt.Sprintf("data/%s/", bot.Config.Name), nil)

	bot.QuoteDB, _ = sql.Open("sqlite3", fmt.Sprintf("data/%s/quotes.db", bot.Config.Name))
	defer bot.QuoteDB.Close()

	// appingdb, err := sql.Open("sqlite3", fmt.Sprintf("data/%s/apping.db", bot.Config.Name))
	// if err != nil {
	// 	log.Panic(err)
	// }
	// defer appingdb.Close()

	// sqlStmt := `create table if not exists fappers (id integer not null primary key, nick text);
	// create table if not exists faps (id integer not null primary key, fapperid integer, starttime timedaye, endtime timedate);`
	// _, err = appingdb.Exec(sqlStmt)
	// if err != nil {
	// 	log.Panic(err)
	// }

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
