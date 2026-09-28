package analytics

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/StdBots/StdQuizBot/internal/credit"
	"github.com/StdBots/StdQuizBot/internal/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var startTime = time.Now()

// Engine renders system health and bot statistics
type Engine struct {
	api     *tgbotapi.BotAPI
	db      *database.DB
	ownerID int64
}

// NewEngine creates analytics engine
func NewEngine(api *tgbotapi.BotAPI, db *database.DB, ownerID int64) *Engine {
	return &Engine{
		api:     api,
		db:      db,
		ownerID: ownerID,
	}
}

// HandleStats renders statistics
func (e *Engine) HandleStats(msg *tgbotapi.Message) {
	if msg.From.ID != e.ownerID {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "⛔ Unauthorized: Admin statistics only.")
		_, _ = e.api.Send(reply)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sysStats, err := e.db.GetSystemStats(ctx)
	if err != nil {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Failed to retrieve system statistics.")
		_, _ = e.api.Send(reply)
		return
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(startTime).Round(time.Second)

	statsText := fmt.Sprintf(
		"📈 <b>StdQuizBot Analytics & Performance</b>\n\n"+
			"👥 <b>Total Users Tracked:</b> %d\n"+
			"🎲 <b>Total Quizzes Created:</b> %d\n"+
			"⚡ <b>Total Quizzes Played:</b> %d\n"+
			"⏱️ <b>Uptime:</b> %s\n\n"+
			"⚙️ <b>System Metrics:</b>\n"+
			"• <b>Go Version:</b> %s\n"+
			"• <b>Goroutines:</b> %d\n"+
			"• <b>Memory Alloc:</b> %.2f MB\n"+
			"• <b>Memory Sys:</b> %.2f MB\n"+
			"• <b>Garbage Collections:</b> %d\n\n"+
			"%s",
		sysStats.TotalUsers,
		sysStats.TotalQuizzes,
		sysStats.TotalPlays,
		uptime,
		runtime.Version(),
		runtime.NumGoroutine(),
		float64(m.Alloc)/(1024*1024),
		float64(m.Sys)/(1024*1024),
		m.NumGC,
		credit.GetFooter(),
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, credit.GetWatermarked(statsText))
	reply.ParseMode = "HTML"

	_, _ = e.api.Send(reply)
}
