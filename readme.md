# News Leeching Telegram Bot

_This bot gets latest news from sources (G.News e.g.), stores them, evaluates them, posts them to a telegram channel and marks as posted._

_Get articles list every 5 minutes and post maximum 5 articles every 30 minutes._

_Identify articles by its guid._

## Prerequisites

### Database

This bot use a MongoDB to store articles (title, url...) and publish them.

Create one with the Database named `newstelegrambot`, and a Collection named `item`.

### Go module

This is written as a go module. So if you want to run it as a standalone program, replace the package to `main`, and the fn `NewsBot` to `main`.

### Telegram

Create a bot, a channel. Add the bot to the channel and grant it a post privilege.

Getting Channel ID: Forward a message of the Chat that we want to get its ID to the @userinfobot

## Instruction

Export environment variable

    export SECRET_TOKEN=your_bot_token
    export CHANNEL_ID=your_channel_id
    export MONGODB_USERNAME=DB_USERNAME
    export MONGODB_PWD=DB_PASSWORD
    export MONGODB_ADDR=DB_ADDRESS
    export MONGODB_PORT=DB_PORT

main.go

```go
import (
	newsbot "github.com/voldedore/NewsTelegramBot"
)

func main() {
	go newsbot.NewsBot()
	select {}
}
```

Build and run.
