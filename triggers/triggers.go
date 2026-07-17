package triggers

import (
	"sync"

	"github.com/Hex-4/bop/ai"
)

type SessionStore struct {
	mu       sync.Mutex
	sessions map[string][]ai.Message
}

func (store *SessionStore) Load(sessionID string) ([]ai.Message, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if messages, ok := store.sessions[sessionID]; ok {
		return messages, nil
	}
	return nil, nil
}

func (store *SessionStore) Save(sessionID string, messages []ai.Message) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.sessions[sessionID] = messages
}
