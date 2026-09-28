package bot

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/StdBots/StdQuizBot/internal/credit"
	"github.com/StdBots/StdQuizBot/internal/database"
	"github.com/StdBots/StdQuizBot/internal/graphics"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ShowGroupLobby initializes the group waiting room
func (b *Bot) ShowGroupLobby(chatID int64, quizID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	quiz, err := b.db.GetQuiz(ctx, quizID)
	if err != nil || len(quiz.Questions) == 0 {
		reply := tgbotapi.NewMessage(chatID, "❌ Quiz not found or has no questions.")
		_, _ = b.api.Send(reply)
		return
	}

	// Check if already running
	existing := b.sessions.GetSession(chatID)
	if existing != nil && existing.Running {
		reply := tgbotapi.NewMessage(chatID, "⚠️ A quiz is already actively running in this group! Send /stop to cancel it.")
		_, _ = b.api.Send(reply)
		return
	}

	sess := b.sessions.StartSession(chatID, quiz, true)

	timerStr := fmt.Sprintf("%d seconds", quiz.Timer)
	if quiz.Timer >= 60 {
		timerStr = fmt.Sprintf("%d minute(s)", quiz.Timer/60)
	}

	text := fmt.Sprintf(
		"♟ Get ready for the quiz *'%s'*\n\n"+
			"✏️ %d question%s\n"+
			"⏱ %s per question\n"+
			"👁 Votes are *visible* to all group members\n\n"+
			"♟ The quiz will begin when at least *2 players* are ready.\n"+
			"Send /stop to cancel anytime.",
		quiz.Title,
		len(quiz.Questions),
		map[bool]string{true: "", false: "s"}[len(quiz.Questions) == 1],
		timerStr,
	)

	keyboard := b.getGroupLobbyKeyboard(quizID, 0)
	msg := tgbotapi.NewMessage(chatID, credit.WatermarkMessage(text))
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	_, _ = b.api.Send(msg)

	_ = sess
}

func (b *Bot) getGroupLobbyKeyboard(quizID string, count int) tgbotapi.InlineKeyboardMarkup {
	readyLabel := "✅ I am ready!"
	if count > 0 {
		readyLabel = fmt.Sprintf("%d ✅ I am ready!", count)
	}

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(readyLabel, fmt.Sprintf("ready_grp:%s", quizID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚀 Start Now (Admin)", fmt.Sprintf("start_grp_now:%s", quizID)),
		),
	)
}

// HandleGroupReady marks a user as ready in group lobby
func (b *Bot) HandleGroupReady(cb *tgbotapi.CallbackQuery, quizID string) {
	chatID := cb.Message.Chat.ID
	user := cb.From

	sess := b.sessions.GetSession(chatID)
	if sess == nil {
		callback := tgbotapi.NewCallback(cb.ID, "Lobby expired. Send /start to begin again.")
		_, _ = b.api.Request(callback)
		return
	}

	sess.mu.Lock()
	if sess.Running {
		sess.mu.Unlock()
		callback := tgbotapi.NewCallback(cb.ID, "Quiz is already running!")
		_, _ = b.api.Request(callback)
		return
	}

	if _, exists := sess.ReadyUsers[user.ID]; exists {
		sess.mu.Unlock()
		callback := tgbotapi.NewCallback(cb.ID, "You are already joined! ✅")
		_, _ = b.api.Request(callback)
		return
	}

	name := user.FirstName
	if name == "" {
		name = user.UserName
	}
	sess.ReadyUsers[user.ID] = name
	count := len(sess.ReadyUsers)
	sess.mu.Unlock()

	callback := tgbotapi.NewCallback(cb.ID, fmt.Sprintf("You're in! (%d players ready)", count))
	_, _ = b.api.Request(callback)

	// Update button count on lobby message
	newKb := b.getGroupLobbyKeyboard(quizID, count)
	edit := tgbotapi.NewEditMessageReplyMarkup(chatID, cb.Message.MessageID, newKb)
	_, _ = b.api.Send(edit)

	// If 2 or more players joined, trigger countdown to start!
	if count >= 2 {
		go func() {
			time.Sleep(2 * time.Second)
			sess.mu.Lock()
			if !sess.Running {
				sess.Running = true
				sess.mu.Unlock()
				b.launchGroupQuiz(sess, cb.Message.MessageID)
			} else {
				sess.mu.Unlock()
			}
		}()
	}
}

// HandleGroupForceStart allows admin or host to start immediately
func (b *Bot) HandleGroupForceStart(cb *tgbotapi.CallbackQuery, quizID string) {
	chatID := cb.Message.Chat.ID
	sess := b.sessions.GetSession(chatID)
	if sess == nil {
		return
	}

	sess.mu.Lock()
	if sess.Running {
		sess.mu.Unlock()
		callback := tgbotapi.NewCallback(cb.ID, "Quiz already running!")
		_, _ = b.api.Request(callback)
		return
	}
	sess.Running = true
	sess.mu.Unlock()

	callback := tgbotapi.NewCallback(cb.ID, "Starting tournament now! 🚀")
	_, _ = b.api.Request(callback)

	go b.launchGroupQuiz(sess, cb.Message.MessageID)
}

func (b *Bot) launchGroupQuiz(sess *ActiveSession, lobbyMsgID int) {
	// Announce tournament launch
	countdownMsg := tgbotapi.NewMessage(sess.ChatID, "🚀 *TOURNAMENT STARTING!* Get ready for Question 1 in 3 seconds...")
	countdownMsg.ParseMode = "Markdown"
	_, _ = b.api.Send(countdownMsg)

	time.Sleep(3 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	quiz, err := b.db.GetQuiz(ctx, sess.QuizID)
	if err != nil || len(quiz.Questions) == 0 {
		return
	}

	// Increment play counter
	go b.db.IncrementQuizPlays(context.Background(), quiz.ID)

	b.runGroupLoop(sess, quiz)
}

func (b *Bot) runGroupLoop(sess *ActiveSession, quiz *database.Quiz) {
	defer b.sessions.RemoveSession(sess.ChatID)

	questions := make([]database.Question, len(quiz.Questions))
	copy(questions, quiz.Questions)

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
			reply := tgbotapi.NewMessage(sess.ChatID, "⛔️ Quiz tournament stopped.")
			_, _ = b.api.Send(reply)
			return
		default:
		}

		qIdx := i + 1
		options := make([]string, len(q.Options))
		copy(options, q.Options)
		correctIdx := q.CorrectIndex

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

		if q.IntroText != "" {
			intro := tgbotapi.NewMessage(sess.ChatID, q.IntroText)
			_, _ = b.api.Send(intro)
			time.Sleep(800 * time.Millisecond)
		}

		// Dispatch native Telegram Quiz Poll
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
		sess.mu.Unlock()

		b.sessions.MapPoll(pollMsg.Poll.ID, sess.ChatID)

		// Wait for question duration
		timerDuration := time.Duration(quiz.Timer)*time.Second + 1500*time.Millisecond
		select {
		case <-time.After(timerDuration):
			// Time is up, move to next question
		case <-sess.Ctx.Done():
			reply := tgbotapi.NewMessage(sess.ChatID, "⛔️ Quiz tournament stopped.")
			_, _ = b.api.Send(reply)
			return
		}
	}

	// Tournament ended, render podium and final scoreboard
	b.sendGroupTournamentResults(sess, quiz, total)
}

func (b *Bot) sendGroupTournamentResults(sess *ActiveSession, quiz *database.Quiz, total int) {
	sess.mu.RLock()
	var players []*PlayerScoreState
	for _, ps := range sess.Scores {
		players = append(players, ps)
	}
	sess.mu.RUnlock()

	if len(players) == 0 {
		text := fmt.Sprintf("♟ The quiz *'%s'* has finished!\n\nNobody submitted answers.", quiz.Title)
		reply := tgbotapi.NewMessage(sess.ChatID, credit.WatermarkMessage(text))
		reply.ParseMode = "Markdown"
		_, _ = b.api.Send(reply)
		return
	}

	// Sort players by:
	// 1. Total Points descending
	// 2. Correct answers descending
	// 3. Average time ascending
	sort.Slice(players, func(i, j int) bool {
		if players[i].TotalPoints != players[j].TotalPoints {
			return players[i].TotalPoints > players[j].TotalPoints
		}
		if players[i].Correct != players[j].Correct {
			return players[i].Correct > players[j].Correct
		}
		avgI := calculateAverageTime(players[i].AnswerTimes)
		avgJ := calculateAverageTime(players[j].AnswerTimes)
		return avgI < avgJ
	})

	// Prepare data for Pure Go HD Podium Banner
	var podiumPlayers []graphics.PlayerResult
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i, p := range players {
		avg := calculateAverageTime(p.AnswerTimes)
		podiumPlayers = append(podiumPlayers, graphics.PlayerResult{
			Rank:     i + 1,
			Name:     p.Name,
			Username: p.Username,
			Score:    p.TotalPoints,
			Correct:  p.Correct,
			Total:    total,
			AvgTime:  avg,
		})

		// Save score to database
		_ = b.db.SavePlayerScore(ctx, database.PlayerScore{
			QuizID:   quiz.ID,
			ChatID:   sess.ChatID,
			UserID:   p.UserID,
			Username: p.Username,
			Name:     p.Name,
			Score:    p.TotalPoints,
			Correct:  p.Correct,
			Total:    total,
			AvgTime:  avg,
		})
	}

	bannerData := graphics.PodiumData{
		QuizTitle:   quiz.Title,
		TotalQ:      total,
		TotalPlayer: len(players),
		TopPlayers:  podiumPlayers,
		FinishedAt:  time.Now(),
	}

	imgBytes, err := graphics.RenderPodiumBanner(bannerData)

	// Build caption
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🏆 <b>TOURNAMENT PODIUM • %s</b>\n\n", quiz.Title))
	medals := []string{"🥇", "🥈", "🥉"}

	for i, p := range podiumPlayers {
		if i >= 10 {
			break
		}
		medal := fmt.Sprintf("<b>#%d</b>", p.Rank)
		if i < len(medals) {
			medal = medals[i]
		}

		userTag := p.Name
		if p.Username != "" {
			userTag = fmt.Sprintf("<a href=\"https://t.me/%s\">%s</a>", p.Username, p.Name)
		}

		sb.WriteString(fmt.Sprintf("%s %s — <b>%d pts</b> (%d/%d • %.1fs avg)\n",
			medal, userTag, p.Score, p.Correct, total, p.AvgTime,
		))
	}

	sb.WriteString(fmt.Sprintf("\n%s", credit.GetFooter()))

	shareLink := fmt.Sprintf("https://t.me/%s?start=%s", b.api.Self.UserName, quiz.ID)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("🔗 Share Quiz", shareLink),
			tgbotapi.NewInlineKeyboardButtonData("📊 All-Time Stats", fmt.Sprintf("stats:%s", quiz.ID)),
		),
	)

	if err == nil && len(imgBytes) > 0 {
		photo := tgbotapi.NewPhoto(sess.ChatID, tgbotapi.FileBytes{
			Name:  "tournament_podium.png",
			Bytes: imgBytes,
		})
		photo.Caption = credit.WatermarkMessage(sb.String())
		photo.ParseMode = tgbotapi.ModeHTML
		photo.ReplyMarkup = keyboard
		_, _ = b.api.Send(photo)
	} else {
		// Fallback to text message
		msg := tgbotapi.NewMessage(sess.ChatID, credit.WatermarkMessage(sb.String()))
		msg.ParseMode = tgbotapi.ModeHTML
		msg.ReplyMarkup = keyboard
		_, _ = b.api.Send(msg)
	}
}

func calculateAverageTime(times []float64) float64 {
	if len(times) == 0 {
		return 0
	}
	sum := 0.0
	for _, t := range times {
		sum += t
	}
	return sum / float64(len(times))
}
