package newsbot

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mmcdole/gofeed"
	"github.com/robfig/cron"
	tb "gopkg.in/tucnak/telebot.v2"
)

const (
	DB_NAME                          string = "news.db"
	NEWS_SRC_TINHTE_LABEL            string = "tinhte"
	NEWS_SRC_BING_LABEL              string = "bing"
	INITIALIZED_FLAG_FILE_NAME       string = "init.done"
	NEWS_SRC_TINHTE_URL              string = "https://feeds.feedburner.com/tinhte/"
	NEWS_SRC_GOOGLE_NEWS_VN_URL      string = "https://news.google.com/rss?hl=vi&gl=VN&ceid=VN:vi"
	NEWS_SRC_GOOGLE_NEWS_US_URL      string = "https://news.google.com/rss?hl=en&gl=US&ceid=US:en"
	LANG_VI                          string = "vi"
	LANG_EN                          string = "en"
	NEWS_SRC_GOOGLE_NEWS_US_TECH_URL string = "https://news.google.com/news/rss/headlines/section/topic/TECHNOLOGY?hl=en&gl=US&ceid=US:en"
)

func initDB() {
	log.Println("Initializing Database...")
	// Create DB
	log.Println("Removing old DB...")
	os.Remove(DB_NAME)
	db, err := sql.Open("sqlite3", DB_NAME)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Println("Creating tables...")
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
	log.Println("Checking for init flag...")
	if file, err := os.Open(INITIALIZED_FLAG_FILE_NAME); os.IsNotExist(err) {
		file.Close()
		log.Println("Not init yet, creating flag...")

		// Write the flag
		file, err := os.Create(INITIALIZED_FLAG_FILE_NAME)
		if err != nil {
			log.Fatal("Flag create failed")
			return
		}
		file.Close()
		log.Println("Flag created")
		initDB()

	} else {
		log.Println("Flag found")
		log.Println("This seems inited. Checking for existed database...")
		if file, err := os.Open(DB_NAME); os.IsNotExist(err) {
			file.Close()
			log.Println("Database does not exist")
			initDB()
		} else { // This should validate DB, a bit overkill for this little project
			log.Println("Database found")
		}
	}
}

func insertArticle(db *sql.DB, source string, articleUrl string) {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare("insert into news_content (source, url) values(?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	_, err = stmt.Exec(source, articleUrl)
	if err != nil {
		log.Fatal(err)
	}
	tx.Commit()
}

func checkIfRowExists(db *sql.DB, articleUrl string) bool {
	stmt, err := db.Prepare("select count(id) as count from news_content where url = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	var count int
	err = stmt.QueryRow(articleUrl).Scan(&count)
	if err != nil {
		log.Fatal(err)
	}

	return count > 0
}

func makeMessage(title string, link string) string {
	return fmt.Sprintf("%s\n\n%s", title, link)
}

// News Bot
func newsBot() {
	b, err := tb.NewBot(tb.Settings{
		Token:  os.Getenv("NEWS_BOT_SECRET_TOKEN"),
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

	// CRON every 5 min, check for the feed update
	c := cron.New()
	c.AddFunc("0 0/5 * * * *", func() {
		go fetchTinhTeNews(b, channel)
		go fetchGoogleNews(b, channel, NEWS_SRC_GOOGLE_NEWS_VN_URL)
	})

	c.Start()
	b.Start()
}

func fetchTinhTeNews(b *tb.Bot, channel *tb.Chat) {
	log.Println("Fetching news...")
	// Fetch and parse RSS
	// Open DB
	db, err := sql.Open("sqlite3", DB_NAME)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// Instantiate NewsParser
	fp := gofeed.NewParser()
	feed, feedErr := fp.ParseURL(NEWS_SRC_TINHTE_URL)
	if feedErr != nil {
		return
	}
	siteName := feed.Generator
	articles := feed.Items
	for _, item := range articles {
		if !checkIfRowExists(db, item.GUID) {
			log.Println(item.GUID)
			insertArticle(db, siteName, item.GUID)
			b.Send(channel, makeMessage(item.Title, item.GUID))
		}
	}

	log.Println("Fetching done")

}

func fetchGoogleNews(b *tb.Bot, channel *tb.Chat, url string) {
	log.Println("Fetching news...")
	// Fetch and parse RSS
	// Open DB
	db, err := sql.Open("sqlite3", DB_NAME)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// Instantiate NewsParser
	fp := gofeed.NewParser()
	feed, feedErr := fp.ParseURL(url)
	if feedErr != nil {
		return
	}
	siteName := feed.Generator
	articles := feed.Items
	for _, item := range articles {
		if !checkIfRowExists(db, item.Link) {
			log.Println(item.Link)
			insertArticle(db, siteName, item.Link)
			b.Send(channel, makeMessage(item.Title, item.Link))
		}
	}

	log.Println("Fetching done")

}

func NewsBot() {
	go prepDB()
	go newsBot()
}
