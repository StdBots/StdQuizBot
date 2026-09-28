package bot

import (
	"context"
	"sync"
	"time"

	"github.com/StdBots/StdQuizBot/internal/database"
)

// PlayerScoreState tracks in-game live score for a participant
type PlayerScoreState struct {
	UserID      int64
	Username    string
	Name        string
	Correct     int
	Wrong       int
	TotalPoints int
	AnswerTimes []float64
}

// ActiveSession stores the live state of a quiz happening in a group or DM
type ActiveSession struct {
	mu                sync.RWMutex
	ChatID            int64
	QuizID            string
	QuizTitle         string
	IsGroup           bool
	Running           bool
	Ctx               context.Context
	Cancel            context.CancelFunc
	ReadyUsers        map[int64]string // userID -> firstName
	Scores            map[int64]*PlayerScoreState
	CurrentQIndex     int
	CurrentCorrectIdx int
	CurrentPollID     string
	QuestionSentAt    time.Time
	SoloAnswerChan    chan struct{}
}

// CreationState tracks multi-step quiz authoring in private chat
type CreationState struct {
	Step      string // "title", "description", "questions", "rename"
	QuizID    string
	IntroText string
}

// SessionManager manages active game sessions and poll answer mappings
type SessionManager struct {
	mu             sync.RWMutex
	sessions       map[int64]*ActiveSession // chatID -> session
	pollMap        map[string]int64         // pollID -> chatID
	creationStates map[int64]*CreationState // userID -> creation state
}

// NewSessionManager creates a thread-safe session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions:       make(map[int64]*ActiveSession),
		pollMap:        make(map[string]int64),
		creationStates: make(map[int64]*CreationState),
	}
}

// GetCreationState retrieves creation state for user
func (sm *SessionManager) GetCreationState(userID int64) *CreationState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.creationStates[userID]
}

// SetCreationState sets creation state for user
func (sm *SessionManager) SetCreationState(userID int64, state *CreationState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if state == nil {
		delete(sm.creationStates, userID)
	} else {
		sm.creationStates[userID] = state
	}
}

// GetSession retrieves active session for a chat
func (sm *SessionManager) GetSession(chatID int64) *ActiveSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.sessions[chatID]
}

// StartSession creates a new active session
func (sm *SessionManager) StartSession(chatID int64, quiz *database.Quiz, isGroup bool) *ActiveSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// If old session exists, cancel it
	if old, exists := sm.sessions[chatID]; exists && old.Cancel != nil {
		old.Cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	sess := &ActiveSession{
		ChatID:         chatID,
		QuizID:         quiz.ID,
		QuizTitle:      quiz.Title,
		IsGroup:        isGroup,
		Running:        false,
		Ctx:            ctx,
		Cancel:         cancel,
		ReadyUsers:     make(map[int64]string),
		Scores:         make(map[int64]*PlayerScoreState),
		SoloAnswerChan: make(chan struct{}, 1),
	}
	sm.sessions[chatID] = sess
	return sess
}

// RemoveSession cleans up an active session
func (sm *SessionManager) RemoveSession(chatID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sess, exists := sm.sessions[chatID]; exists {
		if sess.Cancel != nil {
			sess.Cancel()
		}
		// Clean up poll mappings for this session
		if sess.CurrentPollID != "" {
			delete(sm.pollMap, sess.CurrentPollID)
		}
		delete(sm.sessions, chatID)
	}
}

// MapPoll links a Telegram Poll ID to its chat session
func (sm *SessionManager) MapPoll(pollID string, chatID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.pollMap[pollID] = chatID
}

// GetChatByPoll retrieves ChatID for a given PollID
func (sm *SessionManager) GetChatByPoll(pollID string) (int64, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	chatID, exists := sm.pollMap[pollID]
	return chatID, exists
}
