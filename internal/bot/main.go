package bot

import (
	"runtime/debug"
	"time"

	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/logger"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	botHandlers "github.com/govdbot/govd/internal/bot/handlers"
)

var allowedUpdates = []string{
	"message",
	"callback_query",
	"inline_query",
	"chosen_inline_result",
	"my_chat_member",
}

func Start() {
	b, err := gotgbot.NewBot(config.Env.BotToken, &gotgbot.BotOpts{
		BotClient: NewBotClient(),
	})
	if err != nil {
		logger.L.Fatalf("failed to create bot: %v", err)
	}
	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(_ *gotgbot.Bot, _ *ext.Context, err error) ext.DispatcherAction {
			logger.L.Errorf("an error occurred while handling update: %v", err)
			return ext.DispatcherActionNoop
		},
		Panic: func(_ *gotgbot.Bot, _ *ext.Context, r any) {
			logger.L.Errorf(
				"panic occurred while handling update: %v\n%s",
				r,
				debug.Stack(),
			)
		},
		MaxRoutines: config.Env.ConcurrentUpdates,
	})
	updater := ext.NewUpdater(dispatcher, nil)

	registerHandlers(dispatcher)
	logger.L.Debugf("starting updates polling. allowed updates: %v", allowedUpdates)

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
		logger.L.Fatalf("failed to start polling: %v", err)
	}

	logger.L.Infof("bot started with username: %s", b.Username)
}

func registerHandlers(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(handlers.NewCommand(
		"start",
		botHandlers.StartHandler,
	))
	dispatcher.AddHandler(handlers.NewMyChatMember(
		nil,
		botHandlers.AddGroupHandler,
	))
}
