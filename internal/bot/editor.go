package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/StdBots/StdQuizBot/internal/credit"
	"github.com/StdBots/StdQuizBot/internal/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// SendQuizCard displays the full quiz card with start, group, share, edit, and stats buttons
func (b *Bot) SendQuizCard(chatID int64, quizID string, editMessage bool, messageID int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	quiz, err := b.db.GetQuiz(ctx, quizID)
	if err != nil {
		reply := tgbotapi.NewMessage(chatID, "❌ Quiz not found.")
		_, _ = b.api.Send(reply)
		return
	}

	timerStr := fmt.Sprintf("%ds", quiz.Timer)
	if quiz.Timer >= 60 {
		timerStr = fmt.Sprintf("%d min", quiz.Timer/60)
	}

	shuffleLabel := "No Shuffle"
	switch quiz.Shuffle {
	case "all":
		shuffleLabel = "Shuffle All"
	case "questions":
		shuffleLabel = "Only Questions"
	case "answers":
		shuffleLabel = "Only Answers"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*%s*\n", quiz.Title))
	if quiz.Description != "" {
		sb.WriteString(fmt.Sprintf("_%s_\n", quiz.Description))
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("✏️ %d question%s · ⏱ %s · 🔀 %s\n\n",
		len(quiz.Questions),
		map[bool]string{true: "", false: "s"}[len(quiz.Questions) == 1],
		timerStr,
		shuffleLabel,
	))
	sb.WriteString(fmt.Sprintf("External sharing link:\nhttps://t.me/%s?start=%s\n\n", b.api.Self.UserName, quiz.ID))
	sb.WriteString(credit.GetFooter())

	keyboard := b.getQuizCardKeyboard(quiz.ID)

	if editMessage && messageID > 0 {
		edit := tgbotapi.NewEditMessageText(chatID, messageID, credit.WatermarkMessage(sb.String()))
		edit.ParseMode = "Markdown"
		edit.ReplyMarkup = &keyboard
		_, err = b.api.Send(edit)
		if err == nil {
			return
		}
	}

	msg := tgbotapi.NewMessage(chatID, credit.WatermarkMessage(sb.String()))
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	_, _ = b.api.Send(msg)
}

func (b *Bot) getQuizCardKeyboard(quizID string) tgbotapi.InlineKeyboardMarkup {
	botUser := b.api.Self.UserName
	startGroupURL := fmt.Sprintf("https://t.me/%s?startgroup=%s", botUser, quizID)

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("▶️ Start this quiz", fmt.Sprintf("play_dm:%s", quizID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("👥 Start quiz in group", startGroupURL),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonSwitchInlineQuery("📤 Share quiz", quizID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Edit quiz", fmt.Sprintf("edit:%s", quizID)),
			tgbotapi.NewInlineKeyboardButtonData("📊 Quiz stats", fmt.Sprintf("stats:%s", quizID)),
		),
	)
}

// SendEditMenu renders options to modify quiz properties
func (b *Bot) SendEditMenu(chatID int64, quizID string, messageID int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	quiz, err := b.db.GetQuiz(ctx, quizID)
	if err != nil {
		return
	}

	text := fmt.Sprintf("✏️ *Editing '%s'*\n(%d questions)\n\nWhat would you like to edit?", quiz.Title, len(quiz.Questions))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Edit Title", fmt.Sprintf("edit_title:%s", quizID)),
			tgbotapi.NewInlineKeyboardButtonData("📝 Edit Description", fmt.Sprintf("edit_desc:%s", quizID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Add Questions", fmt.Sprintf("edit_add:%s", quizID)),
			tgbotapi.NewInlineKeyboardButtonData("🗑 Remove Questions", fmt.Sprintf("edit_rm:%s", quizID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏱ Edit Timer", fmt.Sprintf("edit_timer:%s", quizID)),
			tgbotapi.NewInlineKeyboardButtonData("🔀 Edit Shuffle", fmt.Sprintf("edit_shuffle:%s", quizID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Delete Quiz", fmt.Sprintf("edit_delete:%s", quizID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("↩️ Back to Quiz Card", fmt.Sprintf("card:%s", quizID)),
		),
	)

	edit := tgbotapi.NewEditMessageText(chatID, messageID, credit.WatermarkMessage(text))
	edit.ParseMode = "Markdown"
	edit.ReplyMarkup = &keyboard
	_, _ = b.api.Send(edit)
}

// SendRemoveQuestionsMenu lists individual questions for deletion
func (b *Bot) SendRemoveQuestionsMenu(chatID int64, quizID string, messageID int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	quiz, err := b.db.GetQuiz(ctx, quizID)
	if err != nil || len(quiz.Questions) == 0 {
		return
	}

	text := fmt.Sprintf("🗑 Select a question from *'%s'* to remove:", quiz.Title)

	var rows [][]tgbotapi.InlineKeyboardButton
	for i, q := range quiz.Questions {
		label := fmt.Sprintf("❌ Q%d: %s", i+1, TruncateString(q.Question, 24))
		cbData := fmt.Sprintf("rm_q:%s:%d", quizID, i)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, cbData),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("↩️ Back to Edit Menu", fmt.Sprintf("edit:%s", quizID)),
	))

	keyboard := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
	edit := tgbotapi.NewEditMessageText(chatID, messageID, credit.WatermarkMessage(text))
	edit.ParseMode = "Markdown"
	edit.ReplyMarkup = &keyboard
	_, _ = b.api.Send(edit)
}

// SendStats displays leaderboard for a quiz
func (b *Bot) SendStats(chatID int64, quizID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	quiz, err := b.db.GetQuiz(ctx, quizID)
	if err != nil {
		return
	}

	scores, err := b.db.GetQuizLeaderboard(ctx, quizID, 10)
	if err != nil || len(scores) == 0 {
		reply := tgbotapi.NewMessage(chatID, fmt.Sprintf("📊 No plays or stats recorded yet for *'%s'*. Be the first to play!", quiz.Title))
		reply.ParseMode = "Markdown"
		_, _ = b.api.Send(reply)
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📊 *High Scores for '%s'*\n\n", quiz.Title))
	medals := []string{"🥇", "🥈", "🥉"}

	for i, s := range scores {
		medal := fmt.Sprintf("%d.", i+1)
		if i < len(medals) {
			medal = medals[i]
		}
		sb.WriteString(fmt.Sprintf("%s *%s* — %d/%d (%d pts • %.1fs)\n",
			medal, s.Name, s.Correct, s.Total, s.Score, s.AvgTime,
		))
	}

	sb.WriteString(fmt.Sprintf("\n%s", credit.GetFooter()))

	reply := tgbotapi.NewMessage(chatID, credit.WatermarkMessage(sb.String()))
	reply.ParseMode = "Markdown"
	_, _ = b.api.Send(reply)
}

func (b *Bot) getTimerKeyboard(quizID, contextType string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("10 sec", fmt.Sprintf("time:%s:%s:10", contextType, quizID)),
			tgbotapi.NewInlineKeyboardButtonData("15 sec", fmt.Sprintf("time:%s:%s:15", contextType, quizID)),
			tgbotapi.NewInlineKeyboardButtonData("30 sec", fmt.Sprintf("time:%s:%s:30", contextType, quizID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("45 sec", fmt.Sprintf("time:%s:%s:45", contextType, quizID)),
			tgbotapi.NewInlineKeyboardButtonData("1 min", fmt.Sprintf("time:%s:%s:60", contextType, quizID)),
			tgbotapi.NewInlineKeyboardButtonData("2 min", fmt.Sprintf("time:%s:%s:120", contextType, quizID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("3 min", fmt.Sprintf("time:%s:%s:180", contextType, quizID)),
			tgbotapi.NewInlineKeyboardButtonData("4 min", fmt.Sprintf("time:%s:%s:240", contextType, quizID)),
			tgbotapi.NewInlineKeyboardButtonData("5 min", fmt.Sprintf("time:%s:%s:300", contextType, quizID)),
		),
	)
}

func (b *Bot) getShuffleKeyboard(quizID, contextType string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔀 Shuffle All", fmt.Sprintf("shuf:%s:%s:all", contextType, quizID)),
			tgbotapi.NewInlineKeyboardButtonData("🚫 No Shuffle", fmt.Sprintf("shuf:%s:%s:none", contextType, quizID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❓ Only Questions", fmt.Sprintf("shuf:%s:%s:questions", contextType, quizID)),
			tgbotapi.NewInlineKeyboardButtonData("🔤 Only Answers", fmt.Sprintf("shuf:%s:%s:answers", contextType, quizID)),
		),
	)
}

// TruncateString cuts string to length
func TruncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
