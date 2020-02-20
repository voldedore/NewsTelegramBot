package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	// Dependencies
	_ "github.com/mattn/go-sqlite3"
	"github.com/mmcdole/gofeed"
	tb "gopkg.in/tucnak/telebot.v2"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Declare all needed constants
const (
	dbName                string = "news.db"
	newsSrcTinhTeLabel    string = "tinhte"
	newsSrcBingLabel      string = "bing"
	initFlagFilename      string = "init.done"
	newSrcTinhTeURL       string = "https://feeds.feedburner.com/tinhte/"
	newSrcGoogleVNUrl     string = "https://news.google.com/rss?hl=vi&gl=VN&ceid=VN:vi"
	newSrcGoogleUSUrl     string = "https://news.google.com/rss?hl=en&gl=US&ceid=US:en"
	langVi                string = "vi"
	langEn                string = "en"
	newSrcGoogleUSTechURL string = "https://news.google.com/news/rss/headlines/section/topic/TECHNOLOGY?hl=en&gl=US&ceid=US:en"
)

func getOsEnv(variable string, required bool, defaultVal string) string {
	res := os.Getenv(variable)

	if res == "" {
		if required {
			log.Fatalln("No variable named " + variable + " found")
		} else {
			log.Panicln("No variable named " + variable + " found")
		}
		return defaultVal
	}
	return res
}

//mongodb://<dbuser>:<dbpassword>@ds151012.mlab.com:51012/newstelegrambot
func getDB() *mongo.Collection {

	dbUsername := getOsEnv("MONGODB_USERNAME", true, "")
	dbPassword := getOsEnv("MONGODB_PWD", true, "")
	dbAddress := getOsEnv("MONGODB_ADDR", true, "")
	dbPort := getOsEnv("MONGODB_PORT", true, "27017")

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(
		ctx,
		options.Client().ApplyURI("mongodb://"+dbUsername+":"+dbPassword+
			"@"+dbAddress+":"+dbPort+"/newstelegrambot?retryWrites=false"))
	if err != nil {
		log.Fatal(err)
	}
	collection := client.Database("newstelegrambot").Collection("item")

	log.Println("Connected DB")
	return collection
}

func insertArticle(collection *mongo.Collection, source string, article *gofeed.Item) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	_, err := collection.InsertOne(ctx, bson.M{
		"title":   article.Title,
		"link":    article.Link,
		"guid":    article.GUID,
		"pubDate": article.Published,
		"source":  article.Author,
		"points":  0,
	})
	// id := res.InsertedID
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Insert article with guid " + article.GUID)
}

func checkIfRowExists(collection *mongo.Collection, articleGUID string) bool {
	var result struct {
		Value float64
	}
	filter := bson.M{"guid": articleGUID}
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	err := collection.FindOne(ctx, filter).Decode(&result)

	if err != nil {
		return false
	}

	return true
}

func updateRow(collection *mongo.Collection, articleGUID string) {
	opts := options.FindOneAndUpdate().SetUpsert(false)
	filter := bson.D{{"guid", articleGUID}}
	update := bson.D{{"$inc", bson.D{{"points", 1}}}}
	var updatedDocument bson.M
	err := collection.FindOneAndUpdate(context.TODO(), filter, update, opts).Decode(&updatedDocument)
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in the collection
		if err == mongo.ErrNoDocuments {
			return
		}
		log.Fatal(err)
	}

	log.Println("Update article guid " + articleGUID)
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

	var collection = getDB()

	// CRON every 30 min, check for the feed update
	// c := cron.New()
	// c.AddFunc("0 */20 * * * *", func() {
	fetchGoogleNews(b, channel, newSrcGoogleVNUrl, collection)
	// })

	// c.Start()
	b.Start()
}

// func fetchTinhTeNews(b *tb.Bot, channel *tb.Chat) {
// 	log.Println("Fetching news...")
// 	// Fetch and parse RSS
// 	// Open DB
// 	db, err := sql.Open("sqlite3", dbName)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer db.Close()
// 	// Instantiate NewsParser
// 	fp := gofeed.NewParser()
// 	feed, feedErr := fp.ParseURL(newSrcTinhTeURL)
// 	if feedErr != nil {
// 		return
// 	}
// 	siteName := feed.Generator
// 	articles := feed.Items
// 	for _, item := range articles {
// 		if !checkIfRowExists(db, item.GUID) {
// 			log.Println(item.GUID)
// 			insertArticle(db, siteName, item.GUID)
// 			b.Send(channel, makeMessage(item.Title, item.GUID))
// 		}
// 	}

// 	log.Println("Fetching done")

// }

func fetchGoogleNews(b *tb.Bot, channel *tb.Chat, url string, collection *mongo.Collection) {
	log.Println("Fetching news...")

	// Instantiate NewsParser
	fp := gofeed.NewParser()
	feed, feedErr := fp.ParseURL(url)
	if feedErr != nil {
		return
	}
	siteName := feed.Generator
	articles := feed.Items

	for _, item := range articles {
		if !checkIfRowExists(collection, item.GUID) {
			log.Println(item.Title)
			insertArticle(collection, siteName, item)
		} else {
			updateRow(collection, item.GUID)
		}
	}

	log.Println("Fetching done")

}

// NewsBot runs the bot
func main() {
	newsBot()
}
