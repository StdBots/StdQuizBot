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

// HandleCreationText processes text inputs during quiz creation
func (b *Bot) HandleCreationText(msg *tgbotapi.Message) {
	state := b.sessions.GetCreationState(msg.From.ID)
	if state == nil {
		return
	}

	text := strings.TrimSpace(msg.Text)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch state.Step {
	case "title":
		if text == "" {
			reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Title cannot be empty. Please send a valid quiz title:")
			_, _ = b.api.Send(reply)
			return
		}

		creatorName := msg.From.FirstName
		if msg.From.UserName != "" {
			creatorName = "@" + msg.From.UserName
		}

		quiz, err := b.db.CreateQuiz(ctx, msg.From.ID, creatorName, text)
		if err != nil {
			reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Failed to create quiz. Please try again.")
			_, _ = b.api.Send(reply)
			return
		}

		state.QuizID = quiz.ID
		state.Step = "description"
		b.sessions.SetCreationState(msg.From.ID, state)

		descPrompt := "Good. Now send me a *description* of your quiz.\n\n" +
			"This is optional, you can send /skip if you don't want a description."
		reply := tgbotapi.NewMessage(msg.Chat.ID, descPrompt)
		reply.ParseMode = "Markdown"
		_, _ = b.api.Send(reply)

	case "description":
		_ = b.db.UpdateQuiz(ctx, state.QuizID, map[string]interface{}{"description": text})
		state.Step = "questions"
		b.sessions.SetCreationState(msg.From.ID, state)

		prompt := "Good. Now send me a *Poll* with your first question.\n\n" +
			"⚠️ *Important:* Make sure *Quiz Mode* is turned ON in the poll creator (tap the lightbulb icon in Telegram)!\n\n" +
			"💡 *Tip:* You can also send a text message or image first, which will be shown right before the question."
		reply := tgbotapi.NewMessage(msg.Chat.ID, prompt)
		reply.ParseMode = "Markdown"
		_, _ = b.api.Send(reply)

	case "rename":
		if text == "" {
			reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Title cannot be empty.")
			_, _ = b.api.Send(reply)
			return
		}
		_ = b.db.UpdateQuiz(ctx, state.QuizID, map[string]interface{}{"title": text})
		quizID := state.QuizID
		b.sessions.SetCreationState(msg.From.ID, nil)

		reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ Quiz renamed to *%s*!", text))
		reply.ParseMode = "Markdown"
		_, _ = b.api.Send(reply)

		b.SendQuizCard(msg.Chat.ID, quizID, false, 0)

	case "questions":
		// User sent text before sending the poll (intro text)
		state.IntroText = text
		b.sessions.SetCreationState(msg.From.ID, state)

		reply := tgbotapi.NewMessage(msg.Chat.ID, "✅ Intro text saved! Now send the *Quiz Poll* for this question:")
		reply.ParseMode = "Markdown"
		_, _ = b.api.Send(reply)
	}
}

// HandleCreationPoll processes native Telegram quiz polls sent by the creator
func (b *Bot) HandleCreationPoll(msg *tgbotapi.Message) {
	state := b.sessions.GetCreationState(msg.From.ID)
	if state == nil || state.Step != "questions" {
		return
	}

	poll := msg.Poll
	if poll.Type != "quiz" {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Please send a *Quiz* type poll (not a standard survey poll). Turn on 'Quiz Mode' in the poll creator!")
		reply.ParseMode = "Markdown"
		_, _ = b.api.Send(reply)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var options []string
	for _, opt := range poll.Options {
		options = append(options, opt.Text)
	}

	q := database.Question{
		Question:     poll.Question,
		Options:      options,
		CorrectIndex: poll.CorrectOptionID,
		Explanation:  poll.Explanation,
		IntroText:    state.IntroText,
	}

	err := b.db.PushQuestion(ctx, state.QuizID, q)
	if err != nil {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Failed to save question. Please try again.")
		_, _ = b.api.Send(reply)
		return
	}

	// Clear intro text
	state.IntroText = ""
	b.sessions.SetCreationState(msg.From.ID, state)

	quiz, _ := b.db.GetQuiz(ctx, state.QuizID)
	count := len(quiz.Questions)

	text := fmt.Sprintf(
		"Good. Your quiz *'%s'* now has *%d* question(s).\n\n"+
			"• Send the next question poll (or text/media before it)\n"+
			"• Send /undo to remove the last question\n"+
			"• Send /done when you have finished adding questions",
		quiz.Title, count,
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	_, _ = b.api.Send(reply)
}

// HandleSkip skips the optional description step
func (b *Bot) HandleSkip(msg *tgbotapi.Message) {
	state := b.sessions.GetCreationState(msg.From.ID)
	if state != nil && state.Step == "description" {
		state.Step = "questions"
		b.sessions.SetCreationState(msg.From.ID, state)

		prompt := "Description skipped. Now send me a *Poll* with your first question.\n\n" +
			"⚠️ *Important:* Make sure *Quiz Mode* is ON in the poll creator!\n" +
			"Send /done when you finish adding questions."
		reply := tgbotapi.NewMessage(msg.Chat.ID, prompt)
		reply.ParseMode = "Markdown"
		_, _ = b.api.Send(reply)
	} else {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Nothing to skip right now.")
		_, _ = b.api.Send(reply)
	}
}

// HandleUndo removes the last added question
func (b *Bot) HandleUndo(msg *tgbotapi.Message) {
	state := b.sessions.GetCreationState(msg.From.ID)
	if state == nil || state.Step != "questions" {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Nothing to undo.")
		_, _ = b.api.Send(reply)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ok, _ := b.db.PopQuestion(ctx, state.QuizID)
	if ok {
		quiz, _ := b.db.GetQuiz(ctx, state.QuizID)
		n := len(quiz.Questions)
		reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("↩️ Last question removed. Quiz now has *%d* question(s).\n\nSend next question or /done to finish.", n))
		reply.ParseMode = "Markdown"
		_, _ = b.api.Send(reply)
	} else {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "No questions to undo.")
		_, _ = b.api.Send(reply)
	}
}

// HandleDone completes the questions step and displays the timer selection
func (b *Bot) HandleDone(msg *tgbotapi.Message) {
	state := b.sessions.GetCreationState(msg.From.ID)
	if state == nil || state.Step != "questions" {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "You are not creating a quiz right now.")
		_, _ = b.api.Send(reply)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	quiz, err := b.db.GetQuiz(ctx, state.QuizID)
	if err != nil || len(quiz.Questions) == 0 {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Please add at least 1 question before finishing!")
		_, _ = b.api.Send(reply)
		return
	}

	// Done adding questions, clear creation state
	b.sessions.SetCreationState(msg.From.ID, nil)

	prompt := fmt.Sprintf(
		"🎉 Excellent! *'%s'* has *%d* question(s).\n\n"+
			"Please choose a *time limit* per question.\n"+
			"In groups, the bot will automatically proceed when the timer expires.",
		quiz.Title, len(quiz.Questions),
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, credit.WatermarkMessage(prompt))
	reply.ParseMode = "Markdown"
	reply.ReplyMarkup = b.getTimerKeyboard(quiz.ID, "setup")
	_, _ = b.api.Send(reply)
}
