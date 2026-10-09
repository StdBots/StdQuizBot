package bot

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/StdBots/StdQuizBot/internal/credit"
	"github.com/StdBots/StdQuizBot/internal/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ShowDmReadyScreen prompts player to get ready in private chat
func (b *Bot) ShowDmReadyScreen(chatID int64, quizID string, messageID int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	quiz, err := b.db.GetQuiz(ctx, quizID)
	if err != nil || len(quiz.Questions) == 0 {
		reply := tgbotapi.NewMessage(chatID, "❌ Quiz has no questions!")
		_, _ = b.api.Send(reply)
		return
	}

	timerStr := fmt.Sprintf("%d seconds", quiz.Timer)
	if quiz.Timer >= 60 {
		timerStr = fmt.Sprintf("%d minute(s)", quiz.Timer/60)
	}

	text := fmt.Sprintf(
		"♟ Get ready for the quiz *'%s'*\n\n"+
			"✏️ %d question%s\n"+
			"⏱ %s per question\n\n"+
			"♟ Press the button below when you are ready.\n"+
			"Send /stop to cancel anytime.",
		quiz.Title,
		len(quiz.Questions),
		map[bool]string{true: "", false: "s"}[len(quiz.Questions) == 1],
		timerStr,
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ I am ready!", fmt.Sprintf("ready_dm:%s", quizID)),
		),
	)

	if messageID > 0 {
		edit := tgbotapi.NewEditMessageText(chatID, messageID, credit.WatermarkMessage(text))
		edit.ParseMode = "Markdown"
		edit.ReplyMarkup = &keyboard
		_, err = b.api.Send(edit)
		if err == nil {
			return
		}
	}

	msg := tgbotapi.NewMessage(chatID, credit.WatermarkMessage(text))
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	_, _ = b.api.Send(msg)
}

// StartDmQuiz launches solo quiz execution
func (b *Bot) StartDmQuiz(chatID int64, userID int64, quizID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	quiz, err := b.db.GetQuiz(ctx, quizID)
	if err != nil || len(quiz.Questions) == 0 {
		return
	}

	// Initialize session
	sess := b.sessions.StartSession(chatID, quiz, false)
	sess.Running = true

	// Increment play counter
	go b.db.IncrementQuizPlays(context.Background(), quiz.ID)

	go b.runDmLoop(sess, quiz, userID)
}

func (b *Bot) runDmLoop(sess *ActiveSession, quiz *database.Quiz, userID int64) {
	defer b.sessions.RemoveSession(sess.ChatID)

	questions := make([]database.Question, len(quiz.Questions))
	copy(questions, quiz.Questions)

	// Shuffle questions if configured
	if quiz.Shuffle == "all" || quiz.Shuffle == "questions" {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		r.Shuffle(len(questions), func(i, j int) {
			questions[i], questions[j] = questions[j], questions[i]
		})
	}

	total := len(questions)

	for i, q := range questions {
		select {
		case <-sess.Ctx.Done():
			reply := tgbotapi.NewMessage(sess.ChatID, "⛔️ Quiz stopped.")
			_, _ = b.api.Send(reply)
			return
		default:
		}

		qIdx := i + 1
		options := make([]string, len(q.Options))
		copy(options, q.Options)
		correctIdx := q.CorrectIndex

		// Shuffle options if configured
		if quiz.Shuffle == "all" || quiz.Shuffle == "answers" {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			type optPair struct {
				text string
				orig int
			}
			pairs := make([]optPair, len(options))
			for oi, ot := range options {
				pairs[oi] = optPair{text: ot, orig: oi}
			}
			r.Shuffle(len(pairs), func(a, b int) {
				pairs[a], pairs[b] = pairs[b], pairs[a]
			})
			for ni, p := range pairs {
				options[ni] = p.text
				if p.orig == q.CorrectIndex {
					correctIdx = ni
				}
			}
		}

		// If intro text exists, send it before the poll
		if q.IntroText != "" {
			intro := tgbotapi.NewMessage(sess.ChatID, q.IntroText)
			_, _ = b.api.Send(intro)
			time.Sleep(800 * time.Millisecond)
		}

		// Create native Telegram quiz poll
		pollTitle := fmt.Sprintf("[%d/%d] %s", qIdx, total, q.Question)
		pollConfig := tgbotapi.NewSendPoll(sess.ChatID, pollTitle, options...)
		pollConfig.Type = "quiz"
		pollConfig.CorrectOptionID = int64(correctIdx)
		pollConfig.Explanation = q.Explanation
		pollConfig.OpenPeriod = quiz.Timer
		pollConfig.IsAnonymous = false

		pollMsg, err := b.api.Send(pollConfig)
		if err != nil {
			continue
		}

		sess.mu.Lock()
		sess.CurrentQIndex = qIdx
		sess.CurrentCorrectIdx = correctIdx
		sess.CurrentPollID = pollMsg.Poll.ID
		sess.QuestionSentAt = time.Now()
		// Drain any previous solo answer signal
		select {
		case <-sess.SoloAnswerChan:
		default:
		}
		sess.mu.Unlock()

		b.sessions.MapPoll(pollMsg.Poll.ID, sess.ChatID)

		// Wait for either:
		// 1. User answers (SoloAnswerChan)
		// 2. Timer expires
		// 3. Quiz cancelled
		timerDuration := time.Duration(quiz.Timer) * time.Second
		select {
		case <-sess.SoloAnswerChan:
			// User answered early! Give 1.5s to view explanation animation, then next question
			time.Sleep(1500 * time.Millisecond)
		case <-time.After(timerDuration + 500*time.Millisecond):
			// Time expired
		case <-sess.Ctx.Done():
			reply := tgbotapi.NewMessage(sess.ChatID, "⛔️ Quiz stopped.")
			_, _ = b.api.Send(reply)
			return
		}
	}

	// Send Solo Results
	b.sendDmResults(sess, quiz, total, userID)
}

func (b *Bot) sendDmResults(sess *ActiveSession, quiz *database.Quiz, total int, userID int64) {
	sess.mu.RLock()
	scoreState := sess.Scores[userID]
	sess.mu.RUnlock()

	correct := 0
	wrong := 0
	var answerTimes []float64

	if scoreState != nil {
		correct = scoreState.Correct
		wrong = scoreState.Wrong
		answerTimes = scoreState.AnswerTimes
	}

	avgTime := 0.0
	totalTime := 0.0
	for _, t := range answerTimes {
		totalTime += t
	}
	if len(answerTimes) > 0 {
		avgTime = totalTime / float64(len(answerTimes))
	}

	accuracy := 0
	if total > 0 {
		accuracy = (correct * 100) / total
	}

	speedRating := "⚡ Lightning Fast"
	switch {
	case avgTime > 15:
		speedRating = "🧠 Deep Analytical"
	case avgTime > 6:
		speedRating = "🎯 Steady & Sharp"
	}

	resultText := fmt.Sprintf(
		"🏁 *Quiz Finished: '%s'*\n\n"+
			"🏆 *Your Results:*\n"+
			"• *Score:* %d / %d (%d%%)\n"+
			"• *Accuracy:* %d Correct, %d Wrong\n"+
			"• *Avg Speed:* %.1f sec per question\n"+
			"• *Rating:* %s\n\n"+
			"%s",
		quiz.Title,
		correct, total, accuracy,
		correct, wrong,
		avgTime,
		speedRating,
		credit.GetFooter(),
	)

	// Save score to database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, _ := b.db.EnsureUser(ctx, userID, "", "")
	name := "Fighter"
	if user != nil && user.FirstName != "" {
		name = user.FirstName
	}

	points := correct*100 + int(totalTime)
	_ = b.db.SavePlayerScore(ctx, database.PlayerScore{
		QuizID:   quiz.ID,
		ChatID:   sess.ChatID,
		UserID:   userID,
		Username: user.Username,
		Name:     name,
		Score:    points,
		Correct:  correct,
		Total:    total,
		AvgTime:  avgTime,
	})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Retake Quiz", fmt.Sprintf("play_dm:%s", quiz.ID)),
			tgbotapi.NewInlineKeyboardButtonSwitchInlineQuery("📤 Share Quiz", quiz.ID),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Quiz Stats", fmt.Sprintf("stats:%s", quiz.ID)),
		),
	)

	msg := tgbotapi.NewMessage(sess.ChatID, credit.WatermarkMessage(resultText))
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	_, _ = b.api.Send(msg)
}
