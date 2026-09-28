package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/StdBots/StdQuizBot/internal/credit"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleCallback routes all inline button queries
func (b *Bot) HandleCallback(cb *tgbotapi.CallbackQuery) {
	data := cb.Data
	chatID := cb.Message.Chat.ID
	msgID := cb.Message.MessageID
	userID := cb.From.ID

	switch {
	case data == "menu:main":
		b.handleStart(cb.Message, "")
		b.answerCallback(cb.ID, "")

	case data == "create_quiz":
		b.sessions.SetCreationState(userID, &CreationState{Step: "title"})
		reply := tgbotapi.NewMessage(chatID, "Let's create a new quiz. First, send me the *title* of your quiz (e.g., 'Aptitude Test' or '10 Questions about Science'):")
		reply.ParseMode = "Markdown"
		_, _ = b.api.Send(reply)
		b.answerCallback(cb.ID, "")

	case data == "my_quizzes":
		b.ShowUserQuizzes(chatID, userID, msgID)
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "card:"):
		quizID := strings.TrimPrefix(data, "card:")
		b.SendQuizCard(chatID, quizID, true, msgID)
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "play_dm:"):
		quizID := strings.TrimPrefix(data, "play_dm:")
		b.ShowDmReadyScreen(chatID, quizID, msgID)
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "ready_dm:"):
		quizID := strings.TrimPrefix(data, "ready_dm:")
		b.answerCallback(cb.ID, "Let's begin! 🚀")
		b.StartDmQuiz(chatID, userID, quizID)

	case strings.HasPrefix(data, "ready_grp:"):
		quizID := strings.TrimPrefix(data, "ready_grp:")
		b.HandleGroupReady(cb, quizID)

	case strings.HasPrefix(data, "start_grp_now:"):
		quizID := strings.TrimPrefix(data, "start_grp_now:")
		b.HandleGroupForceStart(cb, quizID)

	case strings.HasPrefix(data, "time:"):
		// time:{context}:{quiz_id}:{seconds}
		parts := strings.Split(data, ":")
		if len(parts) == 4 {
			ctxType := parts[1]
			quizID := parts[2]
			secs, _ := strconv.Atoi(parts[3])

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = b.db.UpdateQuiz(ctx, quizID, map[string]interface{}{"timer": secs})
			cancel()

			if ctxType == "setup" {
				// Next step: Shuffle selection
				prompt := "Great! Now choose whether to *shuffle questions and answer options* for each game:"
				keyboard := b.getShuffleKeyboard(quizID, "setup")
				edit := tgbotapi.NewEditMessageText(chatID, msgID, credit.WatermarkMessage(prompt))
				edit.ParseMode = "Markdown"
				edit.ReplyMarkup = &keyboard
				_, _ = b.api.Send(edit)
			} else {
				b.SendQuizCard(chatID, quizID, true, msgID)
			}
			b.answerCallback(cb.ID, fmt.Sprintf("Timer set to %d seconds!", secs))
		}

	case strings.HasPrefix(data, "shuf:"):
		// shuf:{context}:{quiz_id}:{mode}
		parts := strings.Split(data, ":")
		if len(parts) == 4 {
			quizID := parts[2]
			mode := parts[3]

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = b.db.UpdateQuiz(ctx, quizID, map[string]interface{}{"shuffle": mode})
			cancel()

			b.SendQuizCard(chatID, quizID, true, msgID)
			b.answerCallback(cb.ID, "Shuffle settings saved!")
		}

	case strings.HasPrefix(data, "edit:"):
		quizID := strings.TrimPrefix(data, "edit:")
		b.SendEditMenu(chatID, quizID, msgID)
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "edit_title:"):
		quizID := strings.TrimPrefix(data, "edit_title:")
		b.sessions.SetCreationState(userID, &CreationState{Step: "rename", QuizID: quizID})
		reply := tgbotapi.NewMessage(chatID, "Send me the new title for your quiz:")
		_, _ = b.api.Send(reply)
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "edit_desc:"):
		quizID := strings.TrimPrefix(data, "edit_desc:")
		b.sessions.SetCreationState(userID, &CreationState{Step: "description", QuizID: quizID})
		reply := tgbotapi.NewMessage(chatID, "Send me the new description for your quiz:")
		_, _ = b.api.Send(reply)
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "edit_add:"):
		quizID := strings.TrimPrefix(data, "edit_add:")
		b.sessions.SetCreationState(userID, &CreationState{Step: "questions", QuizID: quizID})
		reply := tgbotapi.NewMessage(chatID, "Send me the next *Quiz Poll* to add to this quiz.\n(Send /done when finished)")
		reply.ParseMode = "Markdown"
		_, _ = b.api.Send(reply)
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "edit_rm:"):
		quizID := strings.TrimPrefix(data, "edit_rm:")
		b.SendRemoveQuestionsMenu(chatID, quizID, msgID)
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "rm_q:"):
		// rm_q:{quiz_id}:{index}
		parts := strings.Split(data, ":")
		if len(parts) == 3 {
			quizID := parts[1]
			idx, _ := strconv.Atoi(parts[2])

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = b.db.DeleteQuestionByIndex(ctx, quizID, idx)
			cancel()

			b.SendRemoveQuestionsMenu(chatID, quizID, msgID)
			b.answerCallback(cb.ID, "Question removed!")
		}

	case strings.HasPrefix(data, "edit_timer:"):
		quizID := strings.TrimPrefix(data, "edit_timer:")
		kb := b.getTimerKeyboard(quizID, "edit")
		edit := tgbotapi.NewEditMessageText(chatID, msgID, "⏱ Choose a new *time limit* per question:")
		edit.ParseMode = "Markdown"
		edit.ReplyMarkup = &kb
		_, _ = b.api.Send(edit)
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "edit_shuffle:"):
		quizID := strings.TrimPrefix(data, "edit_shuffle:")
		kb := b.getShuffleKeyboard(quizID, "edit")
		edit := tgbotapi.NewEditMessageText(chatID, msgID, "🔀 Choose *shuffle mode* for this quiz:")
		edit.ParseMode = "Markdown"
		edit.ReplyMarkup = &kb
		_, _ = b.api.Send(edit)
		b.answerCallback(cb.ID, "")

	case strings.HasPrefix(data, "edit_delete:"):
		quizID := strings.TrimPrefix(data, "edit_delete:")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = b.db.DeleteQuiz(ctx, quizID, userID)
		cancel()

		b.answerCallback(cb.ID, "Quiz deleted successfully!")
		b.ShowUserQuizzes(chatID, userID, msgID)

	case strings.HasPrefix(data, "stats:"):
		quizID := strings.TrimPrefix(data, "stats:")
		b.SendStats(chatID, quizID)
		b.answerCallback(cb.ID, "")

	case data == "help:main":
		b.handleHelp(cb.Message)
		b.answerCallback(cb.ID, "")

	default:
		b.answerCallback(cb.ID, "")
	}
}

func (b *Bot) answerCallback(cbID, text string) {
	cb := tgbotapi.NewCallback(cbID, text)
	_, _ = b.api.Request(cb)
}
