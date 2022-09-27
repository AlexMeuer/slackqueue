package token

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
)

type firestoreStore struct {
	client *firestore.Client
	docRef *firestore.DocumentRef
}

type FirestoreStoreOption func(*firestoreStore)

func WithTokenID(ID string) FirestoreStoreOption {
	return func(s *firestoreStore) {
		s.docRef = s.client.Collection("tokens").Doc(ID)
	}
}

func NewFirestoreStore(client *firestore.Client, opts ...FirestoreStoreOption) *firestoreStore {
	s := &firestoreStore{
		client: client,
		docRef: client.Collection("tokens").Doc("token"),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *firestoreStore) SetToken(ctx context.Context, token string) error {
	_, err := s.docRef.Set(ctx, map[string]interface{}{
		"token": token,
	})
	return err
}

func (s *firestoreStore) GetToken(ctx context.Context) (string, error) {
	doc, err := s.docRef.Get(ctx)
	if err != nil {
		return "", err
	}
	var data map[string]interface{}
	if err := doc.DataTo(&data); err != nil {
		return "", err
	}
	token, ok := data["token"].(string)
	if !ok {
		return "", fmt.Errorf("token is not string: %+v", data["token"])
	}
	return token, nil
}
