// Package bot holds the conversational loop a bot runs. A Session is bound to
// one Bot and reuses the personalagent layer for AI turns, so the bot behaves
// like a conversational agent while persisting everything it does in its own
// folder (profile, activities, response history).
package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/lleitep3/aicockpit/internal/personalagent"
)

// Session is a running bot: it keeps a persona, objectives, activities and a
// loop that reacts to events and messages, streaming AI responses through the
// personalagent layer and recording the results into the bot.
type Session struct {
	botID string
	bot   *Bot
	svc   *Service
	pa    *personalagent.Service

	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex

	request chan *sessionRequest
	done    chan struct{}
	err     error
}

// sessionRequest is a unit of work handed to a running session.
type sessionRequest struct {
	event *BotEvent
	msg   string
}

// NewSession creates and starts a bot session. The profile is loaded from the
// bot's persisted folder and merged with the in-memory Bot so the loop always
// reads the latest configuration.
func NewSession(ctx context.Context, svc *Service, bot *Bot) (*Session, error) {
	if svc == nil {
		return nil, errors.New("bot service is nil")
	}
	if bot == nil {
		return nil, errors.New("bot is nil")
	}
	session := &Session{
		botID:   bot.ID,
		bot:     bot,
		svc:     svc,
		pa:      svc.Service,
		request: make(chan *sessionRequest, 16),
		done:    make(chan struct{}),
	}
	session.ctx, session.cancel = context.WithCancel(ctx)

	// Load the persisted profile so the runtime state is always current.
	if p, err := svc.LoadProfile(ctx, bot.ID); err == nil {
		bot.Name = p.Name
		bot.Handle = p.Handle
		bot.Role = p.Role
		bot.Objective = p.Objective
		bot.Persona = p.Persona
		bot.Skills = p.Skills
		bot.Instructions = p.Instructions
		if p.ProviderID != "" {
			bot.Selection.ProviderID = p.ProviderID
		}
		if p.ModelID != "" {
			bot.Selection.ModelID = p.ModelID
		}
		if p.ReasoningEffort != "" {
			bot.Selection.ReasoningEffort = p.ReasoningEffort
		}
		bot.WorkspaceDir = p.WorkspaceDir
	}

	go session.run()
	return session, nil
}

func (s *Session) run() {
	defer close(s.done)
	for {
		select {
		case <-s.ctx.Done():
			return
		case req, ok := <-s.request:
			if !ok {
				return
			}
			if req.msg != "" {
				_, _ = s.ProcessMessage(s.ctx, req.msg, nil)
			}
			if req.event != nil {
				_ = s.ProcessEvent(s.ctx, req.event)
			}
		}
	}
}

// Stop ends the session, cancelling any running AI turn.
func (s *Session) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

// ProcessMessage hands a user message to the bot, creating a conversation for
// it if needed, then streaming the AI response through the session.
func (s *Session) ProcessMessage(ctx context.Context, content string, attachments []personalagent.Attachment) (turnID string, err error) {
	s.mu.Lock()
	bot := s.bot
	s.mu.Unlock()

	if bot == nil {
		return "", errors.New("bot session is not running")
	}
	if content == "" {
		return "", errors.New("message content is empty")
	}

	// Confirm the conversation exists on the personalagent side before
	// starting a turn; this keeps the bot aligned with the shared store.
	_, err = s.pa.Conversation(bot.ID)
	if err != nil {
		if errors.Is(err, personalagent.ErrNotFound) {
			// No conversation yet for this bot; create a dedicated one.
			_, err = personalagent.CreateConversationForBot(ctx, s.pa, bot.ID, bot.Handle)
			if err != nil {
				return "", fmt.Errorf("create bot conversation: %w", err)
			}
			bot.LastPrompt = content
			if err := s.svc.botStore.Save(ctx, bot); err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}

	s.mu.Lock()
	s.bot.LastPrompt = content
	s.mu.Unlock()

	turnID, err = s.pa.StartTurn(bot.ID, content, attachments)
	if err != nil {
		s.mu.Lock()
		s.bot.LastError = err.Error()
		s.mu.Unlock()
		return "", err
	}

	s.mu.Lock()
	s.bot.LastResponse = &BotResponse{
		Text:       "",
		ModelID:    s.bot.Selection.ModelID,
		ProviderID: s.bot.Selection.ProviderID,
		CreatedAt:  time.Now().UTC(),
	}
	s.mu.Unlock()

	return turnID, nil
}

// ProcessEvent applies an observer event to the bot: activities, reminders and
// any other actionable occurrence the loop should react to.
func (s *Session) ProcessEvent(ctx context.Context, ev *BotEvent) error {
	if ev == nil {
		return errors.New("event is nil")
	}
	s.mu.Lock()
	bot := s.bot
	s.mu.Unlock()

	if bot == nil {
		return errors.New("bot session is not running")
	}

	// Surface the event as a user message the bot can reason about.
	msg := fmt.Sprintf("[%s] %s", ev.Kind, ev.Payload)
	_, err := s.ProcessMessage(ctx, msg, nil)
	if err != nil {
		return fmt.Errorf("process event: %w", err)
	}
	return nil
}

// State returns the current persisted state for the bot's folder.
func (s *Session) State() (*BotState, error) {
	path, err := StatePath(s.botID, s.svc.base)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &BotState{BotID: s.botID, UpdatedAt: time.Now().UTC()}, nil
		}
		return nil, err
	}
	var state BotState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("decode state: %w", err)
	}
	return &state, nil
}

// BotState is the runtime state a bot persists between turns.
type BotState struct {
	BotID     string
	LastTurn  string
	UpdatedAt time.Time
	Completed time.Time
}
