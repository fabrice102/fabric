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

// MsgsByID returns messages stored by a certain ID and a given predicate
func (m *MessageStore) Search(id string, filter func(msg gossip.ReceivedMessage) bool) []gossip.ReceivedMessage {
	m.RLock()
	defer m.RUnlock()
	if msgs, exists := m.m[id]; exists {
		return messages(msgs).Filter(filter)
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

type messages []gossip.ReceivedMessage

func (msgs messages) Filter(f func(msg gossip.ReceivedMessage) bool) []gossip.ReceivedMessage {
	var res []gossip.ReceivedMessage
	for _, msg := range msgs {
		if f(msg) {
			res = append(res, msg)
		}
	}
	return res
}
