package bot

import (
	"math"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandlePollAnswer processes real-time user selections on native quiz polls
func (b *Bot) HandlePollAnswer(answer *tgbotapi.PollAnswer) {
	if answer == nil || len(answer.OptionIDs) == 0 {
		return
	}

	pollID := answer.PollID
	chatID, exists := b.sessions.GetChatByPoll(pollID)
	if !exists {
		return
	}

	sess := b.sessions.GetSession(chatID)
	if sess == nil || !sess.Running {
		return
	}

	user := answer.User
	selectedOpt := int(answer.OptionIDs[0])

	sess.mu.Lock()
	defer sess.mu.Unlock()

	// Verify this answer belongs to the current active question
	if sess.CurrentPollID != pollID {
		return
	}

	isCorrect := selectedOpt == sess.CurrentCorrectIdx
	elapsed := time.Since(sess.QuestionSentAt).Seconds()
	if elapsed < 0.1 {
		elapsed = 0.1
	}

	// Speed-based scoring
	// 100 points for correct answer + up to 50 points speed bonus
	points := 0
	if isCorrect {
		speedBonus := int(math.Max(0, 50.0-(elapsed*2.5)))
		points = 100 + speedBonus
	}

	// Fetch or create player score entry
	scoreState, exists := sess.Scores[user.ID]
	if !exists {
		name := user.FirstName
		if name == "" {
			name = user.UserName
		}
		scoreState = &PlayerScoreState{
			UserID:      user.ID,
			Username:    user.UserName,
			Name:        name,
			Correct:     0,
			Wrong:       0,
			TotalPoints: 0,
			AnswerTimes: []float64{},
		}
		sess.Scores[user.ID] = scoreState
	}

	if isCorrect {
		scoreState.Correct++
	} else {
		scoreState.Wrong++
	}
	scoreState.TotalPoints += points
	scoreState.AnswerTimes = append(scoreState.AnswerTimes, math.Round(elapsed*10)/10)

	// If Solo DM session, signal runner to advance to next question
	if !sess.IsGroup && sess.SoloAnswerChan != nil {
		select {
		case sess.SoloAnswerChan <- struct{}{}:
		default:
		}
	}
}
