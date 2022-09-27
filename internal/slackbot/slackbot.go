package slackbot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/ajg/form"
	"github.com/alexmeuer/slackqueue/internal/queue"
	"github.com/nlopes/slack"
)

const (
	ActionJoin  = "joinQueue"
	ActionLeave = "leaveQueue"
)

func New(accessToken string, store QueueStore) (*bot, error) {
	slackApi := slack.New(accessToken)
	if r, err := slackApi.AuthTest(); err != nil {
		return nil, err
	} else {
		log.Println("[Slack] Authenticated for team", r.Team, "as user", r.User)
	}
	if err := slackApi.SetUserAsActive(); err != nil {
		return nil, err
	}
	return &bot{
		client: slackApi,
		queues: store,
		mux:    sync.Mutex{},
	}, nil
}

func (b *bot) HandleCommand(w http.ResponseWriter, r *http.Request) {
	//! TODO: propagate the request Context through to the slack requests.
	p, err := decodePayload(r)
	if err != nil {
		log.Println("[Slack] Failed to decode payload.", err)
		return
	}

	if p.InteractionJson != "" {
		var i interaction
		if err := json.Unmarshal([]byte(p.InteractionJson), &i); err != nil {
			log.Println("[Slack] Failed to parse interaction json.", err)
			w.WriteHeader(http.StatusBadRequest)
		}
		b.handleInteraction(r.Context(), w, &i)
		return
	}

	log.Println("[Slack] Handling", p.Command, p.Text)
	switch p.Command {
	// TODO: clean up these cases.
	case "/joinq":
		b.mux.Lock()
		defer b.mux.Unlock()
		q, err := b.queues.Enqueue(r.Context(), p.ChannelId, queue.Item{
			UserID:   p.UserId,
			UserName: p.UserName,
		})

		if err != nil {
			b.postEphemeralError(p.ChannelId, p.UserId, p.ChannelName, p.UserName, err)
			return
		}
		if _, _, err = b.client.PostMessage(p.ChannelId, slack.MsgOptionBlocks(b.buildBlocks(p.ChannelId, p.ChannelName, q)...)); err != nil {
			b.postEphemeralError(p.ChannelId, p.UserId, p.ChannelName, p.UserName, err)
		}
	case "/leaveq":
		b.mux.Lock()
		defer b.mux.Unlock()
		q, err := b.queues.Dequeue(r.Context(), p.ChannelId, queue.Item{
			UserID:   p.UserId,
			UserName: p.UserName,
		})
		if err != nil {
			b.postEphemeralError(p.ChannelId, p.UserId, p.ChannelName, p.UserName, err)
			return
		}
		if _, _, err = b.client.PostMessage(p.ChannelId, slack.MsgOptionBlocks(b.buildBlocks(p.ChannelId, p.ChannelName, q)...)); err != nil {
			b.postEphemeralError(p.ChannelId, p.UserId, p.ChannelName, p.UserName, err)
		}
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

func decodePayload(r *http.Request) (p payload, err error) {
	d := form.NewDecoder(r.Body)
	d.IgnoreUnknownKeys(true)
	err = d.Decode(&p)
	return
}

func (b *bot) handleInteraction(ctx context.Context, w http.ResponseWriter, i *interaction) {
	log.Println("[Slack] Received interaction from", i.User.Name)

	if len(i.Actions) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("[Slack] No actions found in interaction.")
		return
	}

	switch a := i.Actions[0]; a.ActionID {
	case ActionJoin:
		log.Println("[Slack]", i.User.Name, "would like to join the queue for", i.Channel.ID)
		b.mux.Lock()
		defer b.mux.Unlock()
		q, err := b.queues.Enqueue(ctx, i.Channel.ID, queue.Item{
			UserID:   i.User.ID,
			UserName: i.User.Name,
		})
		if err != nil {
			b.postEphemeralError(i.Channel.ID, i.User.ID, i.Channel.Name, i.User.Name, err)
			return
		}
		_, _, _, err = b.client.UpdateMessage(i.Channel.ID, i.Container.MessageTs, slack.MsgOptionBlocks(b.buildBlocks(i.Channel.ID, i.Channel.Name, q)...))
		if err == nil {
			b.postEphemeral(i.Channel.ID, i.User.ID, i.Channel.Name, i.User.Name, "You have been added to the queue.")
		} else {
			b.postEphemeralError(i.Channel.ID, i.User.ID, i.Channel.Name, i.User.Name, err)
		}

	case ActionLeave:
		log.Println("[Slack]", i.User.Name, "would like to leave the queue for", i.Channel.ID)
		b.mux.Lock()
		defer b.mux.Unlock()
		q, err := b.queues.Dequeue(ctx, i.Channel.ID, queue.Item{
			UserID:   i.User.ID,
			UserName: i.User.Name,
		})
		if err != nil {
			b.postEphemeralError(i.Channel.ID, i.User.ID, i.Channel.Name, i.User.Name, err)
			return
		}
		_, _, _, err = b.client.UpdateMessage(i.Channel.ID, i.Container.MessageTs, slack.MsgOptionBlocks(b.buildBlocks(i.Channel.ID, i.Channel.Name, q)...))
		if err == nil {
			b.postEphemeral(i.Channel.ID, i.User.ID, i.Channel.Name, i.User.Name, "You have been removed from the queue.")
			if len(q) > 0 {
				b.postEphemeral(i.Channel.ID, q[0].UserID, i.Channel.Name, q[0].UserName, fmt.Sprintf(":fleur_de_lis: @%s, you're up!", q[0].UserName))
			}
		} else {
			b.postEphemeralError(i.Channel.ID, i.User.ID, i.Channel.Name, i.User.Name, err)
		}

	default:
		log.Println("[Slack] Unrecognised action id:", a.ActionID)
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (b *bot) postEphemeralError(channelId, userId, channelName, userName string, err error) {
	log.Println("[Slack]", err)
	b.postEphemeral(channelId, userId, channelName, userName, err.Error())
}

func (b *bot) postEphemeral(channelId, userId, channelName, userName string, msg string) {
	if _, err := b.client.PostEphemeral(channelId, userId, slack.MsgOptionText(msg, false), slack.MsgOptionParse(true)); err != nil {
		log.Println("[Slack] Failed to send message to", userName, "on", channelName, err)
	}
}

func emojiForIndex(i int) string {
	switch i {
	case 0:
		return ":first_place_medal:"
	case 1:
		return ":second_place_medal:"
	case 2:
		return ":third_place_medal:"
	default:
		return ":hourglass:"
	}
}

func (b *bot) buildBlocks(id, name string, q []queue.Item) (blocks []slack.Block) {
	sb := strings.Builder{}
	for i, item := range q {
		sb.WriteString(fmt.Sprintf("%s @%s\n", emojiForIndex(i), item.UserName))
	}
	queueStr := sb.String()
	if len(queueStr) == 0 {
		queueStr = "The queue is empty."
	}

	joinButton := slack.NewButtonBlockElement(ActionJoin, id, slack.NewTextBlockObject(slack.PlainTextType, "Join", true, false))
	leaveButton := slack.NewButtonBlockElement(ActionLeave, id, slack.NewTextBlockObject(slack.PlainTextType, "Leave", true, false))

	blocks = []slack.Block{
		slack.NewSectionBlock(slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("%s Queue", name), false, false), nil, nil),
		slack.NewDividerBlock(),
		slack.NewSectionBlock(slack.NewTextBlockObject(slack.MarkdownType, queueStr, false, false), nil, nil),
		slack.NewActionBlock("actions", joinButton, leaveButton),
		slack.NewContextBlock("context", slack.NewTextBlockObject(slack.MarkdownType, "GPLv3 | Authored by @AlexMeuer | github.com/alexmeuer/slackqueue", false, false)),
	}
	return
}
