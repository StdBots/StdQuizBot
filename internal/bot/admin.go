package bot

import (
	"github.com/StdBots/StdQuizBot/internal/analytics"
	"github.com/StdBots/StdQuizBot/internal/broadcast"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleStats(msg *tgbotapi.Message) {
	engine := analytics.NewEngine(b.api, b.db, b.cfg.OwnerID)
	engine.HandleStats(msg)
}

func (b *Bot) handleBroadcast(msg *tgbotapi.Message) {
	engine := broadcast.NewEngine(b.api, b.db, b.cfg.OwnerID)
	engine.HandleBroadcast(msg)
}
