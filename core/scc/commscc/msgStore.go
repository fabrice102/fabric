package commscc

import (
	"sync"
	"github.com/hyperledger/fabric/protos/gossip"
)

// MessageStore struct that encapsulates
// message storage
type MessageStore struct {
	m map[string][]gossip.ReceivedMessage
	sync.RWMutex
}

// NewMessageStore creates new message store instance
func NewMessageStore() *MessageStore {
	return &MessageStore{m: make(map[string][]gossip.ReceivedMessage)}
}

// MsgsByID returns messages stored by a certain ID, or nil
// if such an ID isn't found
func (m *MessageStore) MsgsByID(id string) []gossip.ReceivedMessage {
	m.RLock()
	defer m.RUnlock()
	if msgs, exists := m.m[id]; exists {
		return msgs
	}
	return nil
}

// Put associates msg with the given ID
func (m *MessageStore) Put(id string, msg gossip.ReceivedMessage) {
	m.Lock()
	defer m.Unlock()
	m.m[id] = append(m.m[id], msg)
}

// Remove removes a message with a given ID
func (m *MessageStore) Remove(id string) {
	m.Lock()
	defer m.Unlock()
	delete(m.m, id)
}
/*
// ToSlice returns a slice backed by the elements
// of the MessageStore
func (m *MessageStore) ToSlice() []gossip.ReceivedMessage {
	m.RLock()
	defer m.RUnlock()
	messages := make([]gossip.ReceivedMessage, len(m.m))
	i := 0
	for _, member := range m.m {
		messages[i] = member
		i++
	}
	return messages
}
*/