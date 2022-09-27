package token

import "context"

type InMemoryStore struct {
	Token string
}

func (s *InMemoryStore) SetToken(ctx context.Context, token string) error {
	s.Token = token
	return nil
}

func (s *InMemoryStore) GetToken(ctx context.Context) (string, error) {
	return s.Token, nil
}
