package bot

import (
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/logger"
	"go.uber.org/zap/exp/zapslog"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/choseninlineresult"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/inlinequery"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
	botHandlers "github.com/govdbot/govd/internal/bot/handlers"
	botSettings "github.com/govdbot/govd/internal/bot/handlers/settings"
)

var allowedUpdates = []string{
	"message",
	"callback_query",
	"inline_query",
	"chosen_inline_result",
	"my_chat_member",
}

func Start() {
	bot := createBot()
	dispatcher := newDispatcher()

	// prometheus monitoring
	go monitorDispatcherBuffer(dispatcher)

	updater := ext.NewUpdater(dispatcher, &ext.UpdaterOpts{
		Logger: slog.New(zapslog.NewHandler(logger.L.Desugar().Core())),
	})

	logger.L.Debugf("starting updates polling. allowed updates: %v", allowedUpdates)
	err := updater.StartPolling(bot, &ext.PollingOpts{
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
		logger.L.Fatalf("failed to start polling: %v", err)
	}

	logger.L.Infof("bot started with username: %s", bot.Username)
}

func createBot() *gotgbot.Bot {
	var b *gotgbot.Bot
	var err error

	for i := range 10 {
		b, err = gotgbot.NewBot(config.Env.BotToken, &gotgbot.BotOpts{
			BotClient: NewBotClient(),
		})
		if err == nil {
			return b
		}
		logger.L.Errorf("failed to create bot (attempt %d/10): %v", i+1, err)
		time.Sleep(3 * time.Second)
	}
	logger.L.Fatalf("failed to create bot: %v", err)
	return nil
}

func newDispatcher() *ext.Dispatcher {
	dp := ext.NewDispatcher(&ext.DispatcherOpts{
		Processor: metricsProcessor{processor: ext.BaseProcessor{}},
		Error: func(_ *gotgbot.Bot, _ *ext.Context, err error) ext.DispatcherAction {
			logger.L.Errorf("an error occurred while handling update: %v", err)
			return ext.DispatcherActionNoop
		},
		Panic: func(_ *gotgbot.Bot, _ *ext.Context, r any) {
			logger.L.Errorf(
				"panic occurred while handling update: %v\n%s",
				r, debug.Stack(),
			)
		},
		UnhandledErrFunc: func(e error) {
			logger.L.Errorf("unhandled error occurred: %v", e)
		},
		MaxRoutines: config.Env.ConcurrentUpdates,
		Logger:      slog.New(zapslog.NewHandler(logger.L.Desugar().Core())),
	})
	return registerHandlers(dp)
}

func registerHandlers(dispatcher *ext.Dispatcher) *ext.Dispatcher {
	// url
	dispatcher.AddHandler(handlers.NewMessage(
		botHandlers.URLFilter,
		botHandlers.URLHandler,
	))

	// inline
	dispatcher.AddHandler(handlers.NewInlineQuery(
		inlinequery.All,
		botHandlers.InlineHandler,
	))
	dispatcher.AddHandler(handlers.NewChosenInlineResult(
		choseninlineresult.All,
		botHandlers.InlineResultHandler,
	))
	dispatcher.AddHandler(handlers.NewCallback(
		callbackquery.Equal("inline:loading"),
		botHandlers.InlineLoadingHandler,
	))

	// start
	dispatcher.AddHandler(handlers.NewCommand(
		"start",
		botHandlers.StartHandler,
	))
	dispatcher.AddHandler(handlers.NewCallback(
		callbackquery.Equal("start"),
		botHandlers.StartHandler,
	))

	// extractors
	dispatcher.AddHandler(handlers.NewCallback(
		callbackquery.Equal("extractors"),
		botHandlers.ExtractorsHandler,
	))

	// added to group
	dispatcher.AddHandler(handlers.NewMyChatMember(
		nil,
		botHandlers.AddedToGroupHandler,
	))

	// settings
	dispatcher.AddHandler(handlers.NewCommand(
		"settings",
		botSettings.SettingsHandler,
	))
	dispatcher.AddHandler(handlers.NewCallback(
		callbackquery.Equal("settings"),
		botSettings.SettingsHandler,
	))
	dispatcher.AddHandler(handlers.NewCallback(
		callbackquery.Prefix("settings.options"),
		botSettings.SettingsOptionsHandler,
	))
	dispatcher.AddHandler(handlers.NewCallback(
		callbackquery.Prefix("settings.toggle"),
		botSettings.SettingsToggleHandler,
	))
	dispatcher.AddHandler(handlers.NewCallback(
		callbackquery.Prefix("settings.select"),
		botSettings.SettingsSelectHandler,
	))
	dispatcher.AddHandler(handlers.NewCallback(
		callbackquery.Prefix("settings.many"),
		botSettings.SettingsManyHandler,
	))

	// other
	dispatcher.AddHandler(handlers.NewCallback(
		callbackquery.Equal("close"),
		botHandlers.CloseHandler,
	))
	dispatcher.AddHandler(handlers.NewCommand(
		"derr",
		botHandlers.DecodeErrorHandler,
	))
	dispatcher.AddHandlerToGroup(handlers.NewMessage(
		message.All,
		botHandlers.OldMessagesHandler,
	), -100)
	dispatcher.AddHandler(handlers.NewCommand(
		"stats",
		botHandlers.StatsHandler,
	))
	dispatcher.AddHandler(handlers.NewCallback(
		callbackquery.Prefix("stats"),
		botHandlers.StatsCallbackHandler,
	))

	// whitelist
	if len(config.Env.Whitelist) > 0 {
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

	return dispatcher
}
