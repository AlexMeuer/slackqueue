package queue

import (
	"context"
	"fmt"
)

type inMemoryStore struct {
	Queues map[string][]Item
}

func NewInMemoryStore() *inMemoryStore {
	return &inMemoryStore{
		Queues: make(map[string][]Item),
	}
}

func (s *inMemoryStore) Enqueue(_ context.Context, ID string, item Item) ([]Item, error) {
	s.Queues[ID] = append(s.Queues[ID], item)
	return s.Queues[ID], nil
}

func (s *inMemoryStore) Dequeue(_ context.Context, ID string, item Item) ([]Item, error) {
	q, ok := s.Queues[ID]
	if !ok {
		return nil, fmt.Errorf("queue not found with id: %s", ID)
	}
	for i, v := range q {
		if v.UserID == item.UserID {
			s.Queues[ID] = append(q[:i], q[i+1:]...)
			return s.Queues[ID], nil
		}
	}
	return nil, fmt.Errorf("user (%s) not found in queue with id: %s", item.UserID, ID)
}
