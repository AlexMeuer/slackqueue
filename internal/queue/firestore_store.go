package queue

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
)

type firestoreStore struct {
	client *firestore.Client
}

func NewFirestoreStore(client *firestore.Client) *firestoreStore {
	return &firestoreStore{
		client: client,
	}
}

func (s *firestoreStore) queueRef(ID string) *firestore.CollectionRef {
	return s.client.Collection(fmt.Sprintf("queues/%s/items", ID))
}

func (s *firestoreStore) getQueue(ctx context.Context, ref *firestore.CollectionRef) ([]Item, error) {
	docs, err := ref.OrderBy("createdAt", firestore.Asc).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	items := make([]Item, len(docs))
	for i, doc := range docs {
		if err := doc.DataTo(&items[i]); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (s *firestoreStore) Enqueue(ctx context.Context, ID string, item Item) ([]Item, error) {
	qRef := s.queueRef(ID)
	_, _, err := qRef.Add(ctx, map[string]interface{}{
		"userID":    item.UserID,
		"userName":  item.UserName,
		"createdAt": firestore.ServerTimestamp,
	})
	if err != nil {
		return nil, err
	}
	return s.getQueue(ctx, qRef)
}

func (s *firestoreStore) Dequeue(ctx context.Context, ID string, item Item) ([]Item, error) {
	qRef := s.queueRef(ID)
	docs, err := qRef.Where("userID", "==", item.UserID).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	for _, doc := range docs {
		if _, err := doc.Ref.Delete(ctx); err != nil {
			return nil, err
		}
	}
	return s.getQueue(ctx, qRef)
}
