package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/StdBots/StdQuizBot/internal/credit"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleCommand processes slash commands
func (b *Bot) HandleCommand(msg *tgbotapi.Message) {
	cmd := strings.ToLower(msg.Command())
	args := strings.TrimSpace(msg.CommandArguments())

	switch cmd {
	case "start":
		b.handleStart(msg, args)
	case "newquiz":
		b.handleNewQuiz(msg)
	case "quizzes", "myquizzes":
		b.handleMyQuizzes(msg)
	case "quiz":
		b.handleQuizCommand(msg, args)
	case "stop":
		b.handleStop(msg)
	case "skip":
		b.HandleSkip(msg)
	case "undo":
		b.HandleUndo(msg)
	case "done":
		b.HandleDone(msg)
	case "help":
		b.handleHelp(msg)
	case "stats":
		b.handleStats(msg)
	case "broadcast":
		b.handleBroadcast(msg)
	}
}

func (b *Bot) handleStart(msg *tgbotapi.Message, args string) {
	chatID := msg.Chat.ID
	isGroup := msg.Chat.IsGroup() || msg.Chat.IsSuperGroup()

	// If argument passed via deeplink
	if args != "" {
		quizID := strings.TrimPrefix(args, "quiz_")
		if isGroup {
			b.ShowGroupLobby(chatID, quizID)
			return
		}
		// In private chat: open quiz card directly
		b.SendQuizCard(chatID, quizID, false, 0)
		return
	}

	if isGroup {
		text := "👋 *StdQuizBot is active in this group!*\n\n" +
			"• To launch a tournament here, send `/quiz <quiz_id>` or use the *'Start quiz in group'* button from private chat.\n" +
			"• Send /stop to abort any running quiz.\n\n" +
			credit.GetFooter()
		reply := tgbotapi.NewMessage(chatID, credit.WatermarkMessage(text))
		reply.ParseMode = "Markdown"
		_, _ = b.api.Send(reply)
		return
	}

	// Normal Private /start
	welcomeText := fmt.Sprintf(
		"🎲 *Welcome to @%s!*\n\n"+
			"The premier Telegram *Quiz & Multiplayer Trivia Tournament Engine*.\n"+
			"Inspired by Telegram's official `@QuizBot` — built in high-performance Go with native quiz polls, speed-based scoring, and HD podium victory banners!\n\n"+
			"✨ *Key Capabilities:*\n"+
			"├ 🎯 *Native Quiz Polls:* Native green checkmark/confetti animations\n"+
			"├ 🎮 *Dual-Mode Gameplay:* Play solo in DM or host group tournaments\n"+
			"├ ⚡ *Speed-Based Points:* Bonus points for lightning-fast answers\n"+
			"├ 🏆 *Pure Go HD Podium:* 1000x640 victory banner for winners\n"+
			"└ ✏️ *Full Quiz Editor:* Custom timers (10s-5m), shuffle & question edit\n\n"+
			"Choose an action below to get started:\n\n"+
			"%s",
		b.api.Self.UserName, credit.GetFooter(),
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Create New Quiz", "create_quiz"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 My Quizzes", "my_quizzes"),
			tgbotapi.NewInlineKeyboardButtonData("📖 Help & Guide", "help:main"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("📢 Updates Channel", "https://t.me/STDBOTS"),
			tgbotapi.NewInlineKeyboardButtonURL("👨‍💻 Developer", "https://deepanshu.in"),
		),
	)

	reply := tgbotapi.NewMessage(chatID, credit.WatermarkMessage(welcomeText))
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = keyboard
	_, _ = b.api.Send(reply)
}

func (b *Bot) handleNewQuiz(msg *tgbotapi.Message) {
	if msg.Chat.IsGroup() || msg.Chat.IsSuperGroup() {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Quizzes can only be created in private chat with the bot. Send /newquiz in my DM!")
		_, _ = b.api.Send(reply)
		return
	}

	b.sessions.SetCreationState(msg.From.ID, &CreationState{Step: "title"})

	text := "Let's create a new quiz. First, send me the *title* of your quiz (e.g., 'World Geography' or 'JavaScript Interview Prep'):"
	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	_, _ = b.api.Send(reply)
}

func (b *Bot) handleMyQuizzes(msg *tgbotapi.Message) {
	if msg.Chat.IsGroup() || msg.Chat.IsSuperGroup() {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Please use /quizzes in private chat with the bot.")
		_, _ = b.api.Send(reply)
		return
	}

	b.ShowUserQuizzes(msg.Chat.ID, msg.From.ID, 0)
}

// ShowUserQuizzes displays creator's quizzes list
func (b *Bot) ShowUserQuizzes(chatID, userID int64, messageID int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	quizzes, err := b.db.GetUserQuizzes(ctx, userID)
	if err != nil || len(quizzes) == 0 {
		text := "You haven't created any quizzes yet.\n\nTap *Create New Quiz* to build your first quiz!"
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("➕ Create New Quiz", "create_quiz"),
				tgbotapi.NewInlineKeyboardButtonData("↩️ Main Menu", "menu:main"),
			),
		)
		if messageID > 0 {
			edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
			edit.ParseMode = "Markdown"
			edit.ReplyMarkup = &keyboard
			_, _ = b.api.Send(edit)
		} else {
			reply := tgbotapi.NewMessage(chatID, text)
			reply.ParseMode = "Markdown"
			reply.ReplyMarkup = keyboard
			_, _ = b.api.Send(reply)
		}
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, q := range quizzes {
		btnText := fmt.Sprintf("🎲 %s (%d Qs)", TruncateString(q.Title, 20), len(q.Questions))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnText, fmt.Sprintf("card:%s", q.ID)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("➕ Create New Quiz", "create_quiz"),
		tgbotapi.NewInlineKeyboardButtonData("↩️ Main Menu", "menu:main"),
	))

	text := fmt.Sprintf("📋 *Your Quizzes (%d):*\nSelect a quiz to play, share, or edit:", len(quizzes))
	keyboard := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}

	if messageID > 0 {
		edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
		edit.ParseMode = "Markdown"
		edit.ReplyMarkup = &keyboard
		_, _ = b.api.Send(edit)
	} else {
		reply := tgbotapi.NewMessage(chatID, text)
		reply.ParseMode = "Markdown"
		reply.ReplyMarkup = keyboard
		_, _ = b.api.Send(reply)
	}
}

func (b *Bot) handleQuizCommand(msg *tgbotapi.Message, quizID string) {
	if quizID == "" {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Usage: `/quiz <quiz_id>`\nExample: `/quiz 48a2bc91`")
		reply.ParseMode = "Markdown"
		_, _ = b.api.Send(reply)
		return
	}

	if msg.Chat.IsGroup() || msg.Chat.IsSuperGroup() {
		b.ShowGroupLobby(msg.Chat.ID, quizID)
	} else {
		b.SendQuizCard(msg.Chat.ID, quizID, false, 0)
	}
}

func (b *Bot) handleStop(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	sess := b.sessions.GetSession(chatID)
	if sess != nil && sess.Running {
		if sess.Cancel != nil {
			sess.Cancel()
		}
		b.sessions.RemoveSession(chatID)
		reply := tgbotapi.NewMessage(chatID, "⛔️ Active quiz session has been cancelled.")
		_, _ = b.api.Send(reply)
	} else {
		reply := tgbotapi.NewMessage(chatID, "No quiz is currently running in this chat.")
		_, _ = b.api.Send(reply)
	}
}

func (b *Bot) handleHelp(msg *tgbotapi.Message) {
	helpText := fmt.Sprintf(
		"📖 *HOW TO USE @%s:*\n\n"+
			"*1. Creating a Quiz (in DM):*\n"+
			"• Send /newquiz and provide a title.\n"+
			"• Send a description or /skip.\n"+
			"• Create and send Telegram Quiz Polls (with Quiz Mode ON).\n"+
			"• Send /undo to revert a question or /done to finish.\n"+
			"• Set time limit (10s to 5m) & shuffle mode.\n\n"+
			"*2. Playing Solo in Private:* \n"+
			"• Tap 'Start this quiz' and 'I am ready!'.\n"+
			"• Answer polls at your own speed with instant progression!\n\n"+
			"*3. Hosting in Groups:*\n"+
			"• Tap 'Start quiz in group' to add bot to any group.\n"+
			"• Wait for players to tap 'I am ready!'.\n"+
			"• Live tournament begins with real-time speed scoring.\n"+
			"• Winners receive an HD Victory Podium banner!\n\n"+
			"*Commands:*\n"+
			"• /newquiz - Create a new quiz\n"+
			"• /quizzes - View & manage your quizzes\n"+
			"• /quiz <id> - Launch a quiz in current chat\n"+
			"• /stop - Stop a running quiz\n\n"+
			"%s",
		b.api.Self.UserName, credit.GetFooter(),
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, credit.WatermarkMessage(helpText))
	reply.ParseMode = "Markdown"
	_, _ = b.api.Send(reply)
}
