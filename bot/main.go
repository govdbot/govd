package bot

import (
	"runtime/debug"
	"time"

	botHandlers "github.com/govdbot/govd/bot/handlers"
	"github.com/govdbot/govd/config"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/choseninlineresult"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/inlinequery"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
	"go.uber.org/zap"
)

var allowedUpdates = []string{
	"message",
	"callback_query",
	"inline_query",
	"chosen_inline_result",
}

func Start() {
	b, err := gotgbot.NewBot(config.Env.BotToken, &gotgbot.BotOpts{
		BotClient: NewBotClient(),
	})
	if err != nil {
		zap.S().Fatalf("failed to create bot: %v", err)
	}
	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(_ *gotgbot.Bot, _ *ext.Context, err error) ext.DispatcherAction {
			zap.S().Errorf("an error occurred while handling update: %v", err)
			return ext.DispatcherActionNoop
		},
		Panic: func(_ *gotgbot.Bot, _ *ext.Context, r any) {
			zap.S().Errorf(
				"panic occurred while handling update: %v\n%s",
				r,
				debug.Stack(),
			)
		},
		MaxRoutines: config.Env.ConcurrentUpdates,
	})
	updater := ext.NewUpdater(dispatcher, nil)
	registerHandlers(dispatcher)
	zap.S().Debugf("starting updates polling. allowed updates: %v", allowedUpdates)
	err = updater.StartPolling(b, &ext.PollingOpts{
		DropPendingUpdates: true,
		GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
			Timeout: 9,
			RequestOpts: &gotgbot.RequestOpts{
				Timeout: time.Second * 10,
			},
			AllowedUpdates: allowedUpdates,
		},
	})
	if err != nil {
		zap.S().Fatalf("failed to start polling: %v", err)
	}
	zap.S().Infof("bot started with username: %s", b.Username)
}

func registerHandlers(dispatcher *ext.Dispatcher) {
	zap.S().Debug("registering handlers")
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
	dispatcher.AddHandler(handlers.NewCommand(
		"silent",
		botHandlers.SilentHandler,
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

	// whitelist handlers
	dispatcher.AddHandlerToGroup(handlers.NewMessage(
		message.All,
		botHandlers.WhitelistHandler,
	), -10)
	dispatcher.AddHandlerToGroup(handlers.NewCallback(
		callbackquery.All,
		botHandlers.WhitelistHandler,
	), -10)
	dispatcher.AddHandlerToGroup(handlers.NewInlineQuery(
		inlinequery.All,
		botHandlers.WhitelistHandler,
	), -10)
}
