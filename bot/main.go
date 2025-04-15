package bot

import (
	"log"
	"os"
	"runtime/debug"
	"strconv"
	"time"

	botHandlers "govd/bot/handlers"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/choseninlineresult"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/inlinequery"
)

var AllowedUpdates = []string{
	"message",
	"callback_query",
	"inline_query",
	"chosen_inline_result",
}

func Start() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatalf("BOT_TOKEN is not provided")
	}
	b, err := gotgbot.NewBot(token, &gotgbot.BotOpts{
		BotClient: NewBotClient(),
	})
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}
	concurrentUpdates, err := strconv.Atoi(os.Getenv("CONCURRENT_UPDATES"))
	if err != nil {
		log.Println("failed to parse CONCURRENT_UPDATES env, using 50")
		concurrentUpdates = 50
	}
	logDispatcherErrors, err := strconv.ParseBool(os.Getenv("LOG_DISPATCHER_ERRORS"))
	if err != nil {
		log.Println("failed to parse LOG_DISPATCHER_ERRORS env, using false")
		logDispatcherErrors = false
	}
	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			if logDispatcherErrors {
				log.Printf("an error occurred while handling update: %v", err)
			}
			return ext.DispatcherActionNoop
		},
		Panic: func(b *gotgbot.Bot, ctx *ext.Context, r interface{}) {
			if logDispatcherErrors {
				log.Printf("panic occurred while handling update: %v\n", r)
				log.Printf("stack trace:\n%s\n", debug.Stack())
			}
		},
		MaxRoutines: concurrentUpdates,
	})
	updater := ext.NewUpdater(dispatcher, nil)
	registerHandlers(dispatcher)
	err = updater.StartPolling(b, &ext.PollingOpts{
		DropPendingUpdates: true,
		GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
			Timeout: 9,
			RequestOpts: &gotgbot.RequestOpts{
				Timeout: time.Second * 10,
			},
			AllowedUpdates: AllowedUpdates,
		},
	})
	if err != nil {
		log.Fatalf("failed to start polling: %v", err)
	}
	log.Printf("bot started on: %s\n", b.User.Username)
}

func registerHandlers(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(handlers.NewMessage(
		botHandlers.URLFilter,
		botHandlers.URLHandler,
	))
	dispatcher.AddHandler(handlers.NewCommand(
		"start",
		botHandlers.StartHandler,
	))
	dispatcher.AddHandler(handlers.NewCallback(
		callbackquery.Equal("start"),
		botHandlers.StartHandler,
	))
	dispatcher.AddHandler(handlers.NewCallback(
		callbackquery.Equal("help"),
		botHandlers.HelpHandler,
	))
	dispatcher.AddHandler(handlers.NewCommand(
		"settings",
		botHandlers.SettingsHandler,
	))
	dispatcher.AddHandler(handlers.NewCommand(
		"captions",
		botHandlers.CaptionsHandler,
	))
	dispatcher.AddHandler(handlers.NewCommand(
		"nsfw",
		botHandlers.NSFWHandler,
	))
	dispatcher.AddHandler(handlers.NewCommand(
		"limit",
		botHandlers.MediaGroupLimitHandler,
	))
	dispatcher.AddHandler(handlers.NewCallback(
		callbackquery.Equal("stats"),
		botHandlers.StatsHandler,
	))
	dispatcher.AddHandler(handlers.NewCallback(
		callbackquery.Equal("extractors"),
		botHandlers.ExtractorsHandler,
	))
	dispatcher.AddHandler(handlers.NewCallback(
		callbackquery.Equal("instances"),
		botHandlers.InstancesHandler,
	))
	dispatcher.AddHandler(handlers.NewInlineQuery(
		inlinequery.All,
		botHandlers.InlineDownloadHandler,
	))
	dispatcher.AddHandler(handlers.NewChosenInlineResult(
		choseninlineresult.All,
		botHandlers.InlineDownloadResultHandler,
	))
	dispatcher.AddHandler(handlers.NewCallback(
		callbackquery.Equal("inline:loading"),
		botHandlers.InlineLoadingHandler,
	))
}
