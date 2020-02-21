package newsbot

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	// Dependencies
	"github.com/mmcdole/gofeed"
	"github.com/robfig/cron"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	tb "gopkg.in/tucnak/telebot.v2"
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
	// voteInterval is the interval of time that we vote the list of article
	voteInterval int = 5
	// publishInterval is the interval of time that we publish the articles if it pass the minimum
	// points
	publishInterval int = 30
	// Threshold is the minimum point of the article be published, in ${voteInterval} minutes
	// This is calculated by publishInterval / voteInterval
	threshold int = 6
	// Limit is the maximum qty of item to be publish at once
	limit int64 = 5
)

// ArticleItem describe what should have in an article entity
type ArticleItem struct {
	ID      *primitive.ObjectID `json:"ID" bson:"_id,omitempty"`
	Name    string              `json:"name" bson:"name"`
	Title   string              `json:"title" bson:"title"`
	Link    string              `json:"link" bson:"link"`
	GUID    string              `json:"guid" bson:"guid"`
	PubDate string              `json:"pubDate" bson:"pubDate"`
	Source  string              `json:"source" bson:"source"`
	Points  int                 `json:"points" bson:"points"`
	Publish bool                `json:"publish" bson:"publish"`
}

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
		"publish": false,
	})
	// id := res.InsertedID
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Insert article guid " + article.GUID)
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

func markAsPublished(collection *mongo.Collection, articleGUID string) {
	opts := options.FindOneAndUpdate().SetUpsert(false)
	filter := bson.D{{"guid", articleGUID}}
	update := bson.D{{"$set", bson.D{{"publish", true}}}}
	var updatedDocument bson.M
	err := collection.FindOneAndUpdate(context.TODO(), filter, update, opts).Decode(&updatedDocument)
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in the collection
		if err == mongo.ErrNoDocuments {
			return
		}
		log.Fatal(err)
	}

	log.Println("Published article guid " + articleGUID)
}

func publish(collection *mongo.Collection, b *tb.Bot, channel *tb.Chat) {
	log.Println("Publishing...")
	opts := options.Find().SetSort(bson.D{{"points", 1}}).SetLimit(limit)
	cursor, err := collection.Find(context.TODO(), bson.D{{"publish", false}, {"points", bson.D{{"$gt", 5}}}}, opts)
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(context.TODO())

	// Iterate the cursor and print out each document until the cursor is exhausted or there is an error getting the
	// next document.
	item := ArticleItem{}
	for cursor.Next(context.TODO()) {
		if err := cursor.Decode(&item); err != nil {
			log.Fatal(err)
		}
		log.Println("Publish:" + makeMessage(item.Title, item.Link))
		markAsPublished(collection, item.GUID)
		b.Send(channel, makeMessage(item.Title, item.Link), tb.ModeMarkdown)
	}
	if err := cursor.Err(); err != nil {
		log.Fatal(err)
	}

	log.Println("End publishing...")
}

func makeMessage(title string, link string) string {
	return fmt.Sprintf("[%s](%s)", title, link)
}

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

	// Cron
	// The legacy syntax (asterisks) doesn't work if this package is imported to another program
	// Although it runs well directly if this package was compile as a complete program
	c := cron.New()
	c.AddFunc("@every "+strconv.Itoa(voteInterval)+"m", func() {
		fetchGoogleNews(b, channel, newSrcGoogleVNUrl, collection)
	})

	c.AddFunc("@every "+strconv.Itoa(publishInterval)+"m", func() {
		publish(collection, b, channel)
	})

	c.Start()
	// Testing purpose
	// publish(collection, b, channel)
	b.Start()
}

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

// NewsBot starts the bot, connects to DB, starts the cron
func NewsBot() {
	newsBot()
}
