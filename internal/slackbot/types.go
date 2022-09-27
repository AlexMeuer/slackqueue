package slackbot

import (
	"context"
	"sync"

	"github.com/alexmeuer/slackqueue/internal/queue"
	"github.com/nlopes/slack"
)

type QueueStore interface {
	Enqueue(ctx context.Context, ID string, item queue.Item) ([]queue.Item, error)
	Dequeue(ctx context.Context, ID string, item queue.Item) ([]queue.Item, error)
}

type TokenStore interface {
	GetToken(ctx context.Context) (string, error)
	SetToken(ctx context.Context, token string) error
}

type bot struct {
	client       *slack.Client
	clientID     string
	clientSecret string
	tknStore     TokenStore
	queues       QueueStore
	mux          sync.Mutex
}

type payload struct {
	TeamId         string `form:"team_id"`
	TeamDomain     string `form:"team_domain"`
	EnterpriseId   string `form:"enterprise_id"`
	EnterpriseName string `form:"enterprise_name"`
	ChannelId      string `form:"channel_id"`
	ChannelName    string `form:"channel_name"`
	UserId         string `form:"user_id"`
	UserName       string `form:"user_name"`
	Command        string `form:"command"`
	Text           string `form:"text"`
	ResponseUrl    string `form:"response_url"`
	TriggerId      string `form:"trigger_id"`

	// Only present on payloads for interactions.
	InteractionJson string `form:"payload"`
}

type interaction struct {
	Type string `json:"type"`
	Team struct {
		ID     string `json:"id"`
		Domain string `json:"domain"`
	} `json:"team"`
	User struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Name     string `json:"name"`
	} `json:"user"`
	APIAppID  string `json:"api_app_id"`
	Token     string `json:"token"`
	Container struct {
		Type        string `json:"type"`
		MessageTs   string `json:"message_ts"`
		ChannelID   string `json:"channel_id"`
		IsEphemeral bool   `json:"is_ephemeral"`
	} `json:"container"`
	TriggerID string `json:"trigger_id"`
	Channel   struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"channel"`
	ResponseURL string `json:"response_url"`
	Actions     []struct {
		ActionID string `json:"action_id"`
		BlockID  string `json:"block_id"`
		Text     struct {
			Type  string `json:"type"`
			Text  string `json:"text"`
			Emoji bool   `json:"emoji"`
		} `json:"text"`
		Value    string `json:"value"`
		Type     string `json:"type"`
		ActionTs string `json:"action_ts"`
	} `json:"actions"`
}
