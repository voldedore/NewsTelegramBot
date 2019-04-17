package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	tb "gopkg.in/tucnak/telebot.v2"
)

const DB_NAME string = "news.db"
const NEWS_SRC_TINHTE_LABEL string = "tinhte"
const NEWS_SRC_BING_LABEL string = "bing"
const INITIALIZED_FLAG_FILE_NAME string = "init.done"

func initDB() {
	// Create DB
	os.Remove(DB_NAME)
	db, err := sql.Open("sqlite3", DB_NAME)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sqlStmt := `
    create table news_content (id integer not null primary key, source text, url text);
    `
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}
}

func prepDB() {

	// Try to open the flag, if it's not exist, then init the file
	if file, err := os.Open(INITIALIZED_FLAG_FILE_NAME); os.IsNotExist(err) {
		file.Close()

		// Write the flag
		file, err := os.Create(INITIALIZED_FLAG_FILE_NAME)
		if err != nil {
			log.Fatal("Flag create failed")
			return
		}
		file.Close()
		initDB()

	} else {
		log.Println("This seems inited")
		if file, err := os.Open(DB_NAME); os.IsNotExist(err) {
			file.Close()
			initDB()
		} // This should validate DB, a bit overkill for this little project
	}
}

// News Bot
func newsBot() {
	b, err := tb.NewBot(tb.Settings{
		Token:  os.Getenv("SECRET_TOKEN"),
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatal(err)
		return
	}

	b.Handle(tb.OnText, func(m *tb.Message) {
		b.Send(m.Sender, "This bot does not currently support the interactive mode")
	})

	// Getting the Channel
	channel, channelGetErr := b.ChatByID(os.Getenv("CHANNEL_ID"))

	if channelGetErr != nil {
		log.Fatal(channelGetErr)
		return
	}

	b.Send(channel, "This test message")

	b.Start()
}

func main() {
	go prepDB()
	go newsBot()
	select {}
}
