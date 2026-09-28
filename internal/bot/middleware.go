package bot

import (
	"log"
	"runtime/debug"
)

// SafeHandle wraps operations in defer-recover to prevent crash
func (b *Bot) SafeHandle(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC RECOVERED] %v\nStack: %s", r, debug.Stack())
		}
	}()
	fn()
}

// IsOwner checks if user ID matches bot owner
func (b *Bot) IsOwner(userID int64) bool {
	return userID == b.cfg.OwnerID
}
