package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/StdBots/StdQuizBot/internal/config"
	"github.com/StdBots/StdQuizBot/internal/credit"
	"github.com/StdBots/StdQuizBot/internal/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot encapsulates telegram API and dependencies
type Bot struct {
	api       *tgbotapi.BotAPI
	db        *database.DB
	cfg       *config.Config
	sessions  *SessionManager
	startTime time.Time
}

// New creates and initializes Bot instance
func New(cfg *config.Config, db *database.DB) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, err
	}

	if cfg.Env == "development" {
		api.Debug = true
	}

	log.Printf("[BOT] Authorized on account @%s (ID: %d)", api.Self.UserName, api.Self.ID)

	b := &Bot{
		api:       api,
		db:        db,
		cfg:       cfg,
		sessions:  NewSessionManager(),
		startTime: time.Now(),
	}

	b.registerBotCommands()

	// Verify credit integrity
	ok, tampered := credit.VerifyIntegrity()
	if !ok {
		log.Printf("[WARN] Credit verification issues detected: %v", tampered)
	}

	credit.ReportForkStatus(api.Self.UserName, ok)

	return b, nil
}

func (b *Bot) registerBotCommands() {
	commands := []tgbotapi.BotCommand{
		{Command: "newquiz", Description: "Create a new quiz with questions"},
		{Command: "quizzes", Description: "List and manage your created quizzes"},
		{Command: "quiz", Description: "Launch a quiz in the chat (/quiz <id>)"},
		{Command: "stop", Description: "Stop an active running quiz session"},
		{Command: "help", Description: "Guide on how to play, host & create"},
	}

	cmdConfig := tgbotapi.NewSetMyCommands(commands...)
	_, err := b.api.Request(cmdConfig)
	if err != nil {
		log.Printf("[WARN] Failed to set bot commands: %v", err)
	}
}

// Start begins update polling loop
func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := b.api.GetUpdatesChan(u)
	log.Printf("[BOT] @%s listening for quiz interactions & polls...", b.api.Self.UserName)

	for update := range updates {
		up := update
		go b.SafeHandle(func() {
			b.processUpdate(up)
		})
	}
}

func (b *Bot) processUpdate(update tgbotapi.Update) {
	// 1. Handle Inline Keyboards
	if update.CallbackQuery != nil {
		b.HandleCallback(update.CallbackQuery)
		return
	}

	// 2. Handle Real-Time Native Quiz Poll Answers
	if update.PollAnswer != nil {
		b.HandlePollAnswer(update.PollAnswer)
		return
	}

	// 3. Handle Inline Share Queries
	if update.InlineQuery != nil {
		b.handleInlineQuery(update.InlineQuery)
		return
	}

	// 4. Handle Messages
	if update.Message != nil {
		msg := update.Message

		// Track user in database
		if msg.From != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_, _ = b.db.EnsureUser(ctx, msg.From.ID, msg.From.UserName, msg.From.FirstName)
			}()
		}

		if msg.IsCommand() {
			b.HandleCommand(msg)
			return
		}

		// Quiz Poll sent by user
		if msg.Poll != nil {
			b.HandleCreationPoll(msg)
			return
		}

		// Text message
		if msg.Text != "" {
			if msg.Chat.IsPrivate() {
				state := b.sessions.GetCreationState(msg.From.ID)
				if state != nil {
					b.HandleCreationText(msg)
					return
				}
				b.handleStart(msg, "")
			}
		}
	}
}

// handleInlineQuery enables inline quiz sharing across Telegram chats
func (b *Bot) handleInlineQuery(iq *tgbotapi.InlineQuery) {
	query := strings.TrimSpace(iq.Query)
	if query == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	quiz, err := b.db.GetQuiz(ctx, query)
	if err != nil {
		return
	}

	desc := fmt.Sprintf("%d questions • %ds timer", len(quiz.Questions), quiz.Timer)
	if quiz.Description != "" {
		desc = quiz.Description
	}

	article := tgbotapi.NewInlineQueryResultArticle(
		quiz.ID,
		fmt.Sprintf("🎲 Quiz: %s", quiz.Title),
		fmt.Sprintf("🎲 *%s*\n_%s_\n\nTap below to start this quiz!", quiz.Title, desc),
	)
	article.Description = desc

	startURL := fmt.Sprintf("https://t.me/%s?start=%s", b.api.Self.UserName, quiz.ID)
	startGroupURL := fmt.Sprintf("https://t.me/%s?startgroup=%s", b.api.Self.UserName, quiz.ID)

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("▶️ Play Solo", startURL),
			tgbotapi.NewInlineKeyboardButtonURL("👥 Play in Group", startGroupURL),
		),
	)
	article.ReplyMarkup = &kb

	inlineConf := tgbotapi.InlineConfig{
		InlineQueryID: iq.ID,
		Results:       []interface{}{article},
		CacheTime:     10,
	}
	_, _ = b.api.Request(inlineConf)
}
