package tui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/lleitep3/aicockpit/internal/personalagent"
	"golang.org/x/term"
)

type tuiMode uint8

const (
	tuiList tuiMode = iota
	tuiSearch
	tuiConversation
	tuiCompose
)

type terminalUI struct {
	ctx           context.Context
	service       chatService
	input         *os.File
	output        io.Writer
	keys          <-chan byte
	mode          tuiMode
	conversations []personalagent.ConversationSummary
	selected      int
	search        string
	composer      string
	conversation  *personalagent.Conversation
	streaming     string
	status        string
}

type chatService interface {
	ConversationSummaries() ([]personalagent.ConversationSummary, error)
	Conversations() ([]personalagent.Conversation, error)
	Conversation(id string) (personalagent.Conversation, error)
	CreateConversation(
		ctx context.Context,
		providerID, modelID, reasoningEffort, approvalPolicy, routingMode string,
	) (personalagent.Conversation, error)
	StartTurn(conversationID, content string, attachments []personalagent.Attachment) (string, error)
	Subscribe(turnID string, after uint64) ([]personalagent.TurnEvent, <-chan personalagent.TurnEvent, func())
	DecideApproval(conversationID, turnID, requestID, decision string) error
	CancelTurn(conversationID, turnID string) error
}

func RunTUI(ctx context.Context, service chatService, input *os.File, output io.Writer) error {
	if !term.IsTerminal(int(input.Fd())) {
		return errors.New("interactive terminal required; run `cockpit agent --tui` in a terminal")
	}
	state, err := term.MakeRaw(int(input.Fd()))
	if err != nil {
		return fmt.Errorf("enable terminal input mode: %w", err)
	}
	defer func() {
		_ = term.Restore(int(input.Fd()), state)
		_, _ = io.WriteString(output, "\x1b[0m\x1b[?25h\n")
	}()

	ui := &terminalUI{
		ctx:     ctx,
		service: service,
		input:   input,
		output:  output,
		keys:    readTerminalKeys(ctx, input),
	}
	return ui.run()
}

func (ui *terminalUI) run() error {
	if err := ui.refreshConversations(); err != nil {
		return err
	}
	for {
		if err := ui.draw(); err != nil {
			return err
		}
		select {
		case <-ui.ctx.Done():
			return nil
		case key, open := <-ui.keys:
			if !open {
				return nil
			}
			done, err := ui.handleKey(key)
			if err != nil {
				ui.status = err.Error()
			}
			if done {
				return nil
			}
		}
	}
}

func readTerminalKeys(ctx context.Context, input *os.File) <-chan byte {
	keys := make(chan byte, 16)
	go func() {
		defer close(keys)
		var value [1]byte
		for {
			if _, err := input.Read(value[:]); err != nil {
				return
			}
			select {
			case keys <- value[0]:
			case <-ctx.Done():
				return
			}
		}
	}()
	return keys
}

func (ui *terminalUI) handleKey(key byte) (bool, error) {
	switch ui.mode {
	case tuiList:
		return ui.handleListKey(key)
	case tuiSearch:
		return false, ui.handleSearchKey(key)
	case tuiConversation:
		return ui.handleConversationKey(key)
	case tuiCompose:
		return false, ui.handleComposeKey(key)
	default:
		return false, errors.New("unknown terminal interface state")
	}
}

func (ui *terminalUI) handleListKey(key byte) (bool, error) {
	switch key {
	case 'q', 3:
		return true, nil
	case 'j':
		if ui.selected < len(ui.conversations)-1 {
			ui.selected++
		}
	case 'k':
		if ui.selected > 0 {
			ui.selected--
		}
	case '/':
		ui.mode = tuiSearch
		ui.composer = ""
	case 'n':
		ui.conversation = nil
		ui.streaming = ""
		ui.mode = tuiConversation
		ui.status = "Draft chat; press i to write. Nothing saved until first send."
	case '\r', '\n':
		if len(ui.conversations) == 0 {
			return false, nil
		}
		conversation, err := ui.service.Conversation(ui.conversations[ui.selected].ID)
		if err != nil {
			return false, fmt.Errorf("open conversation: %w", err)
		}
		ui.conversation = &conversation
		ui.mode = tuiConversation
		ui.status = ""
	}
	return false, nil
}

func (ui *terminalUI) handleSearchKey(key byte) error {
	switch key {
	case 3, 27:
		ui.mode = tuiList
		ui.composer = ""
	case '\r', '\n':
		ui.search = strings.TrimSpace(ui.composer)
		ui.composer = ""
		ui.mode = tuiList
		ui.selected = 0
		return ui.refreshConversations()
	case 8, 127:
		ui.composer = removeLastRune(ui.composer)
	default:
		if key >= 32 {
			ui.composer += string([]byte{key})
		}
	}
	return nil
}

func (ui *terminalUI) handleConversationKey(key byte) (bool, error) {
	switch key {
	case 'q', 3:
		return true, nil
	case 'b':
		ui.conversation = nil
		ui.streaming = ""
		ui.mode = tuiList
		return false, ui.refreshConversations()
	case 'i':
		ui.composer = ""
		ui.mode = tuiCompose
		ui.status = "Type message; Enter sends, Ctrl+J adds a line, Ctrl+C cancels."
	}
	return false, nil
}

func (ui *terminalUI) handleComposeKey(key byte) error {
	switch key {
	case 3:
		ui.composer = ""
		ui.mode = tuiConversation
		ui.status = "Message cancelled."
	case 21:
		ui.composer = ""
	case 10:
		ui.composer += "\n"
	case '\r':
		return ui.sendMessage()
	case 8, 127:
		ui.composer = removeLastRune(ui.composer)
	default:
		if key >= 32 {
			ui.composer += string([]byte{key})
		}
	}
	return nil
}

func (ui *terminalUI) refreshConversations() error {
	summaries, err := ui.service.ConversationSummaries()
	if err != nil {
		return fmt.Errorf("load conversations: %w", err)
	}
	records := make(map[string]personalagent.Conversation)
	if ui.search != "" {
		conversations, err := ui.service.Conversations()
		if err != nil {
			return fmt.Errorf("search conversations: %w", err)
		}
		for _, conversation := range conversations {
			records[conversation.ID] = conversation
		}
	}
	ui.conversations = filterTUIConversations(summaries, records, ui.search)
	if ui.selected >= len(ui.conversations) {
		ui.selected = len(ui.conversations) - 1
	}
	if ui.selected < 0 {
		ui.selected = 0
	}
	return nil
}

func filterTUIConversations(
	summaries []personalagent.ConversationSummary,
	records map[string]personalagent.Conversation,
	query string,
) []personalagent.ConversationSummary {
	query = strings.ToLower(strings.TrimSpace(query))
	filtered := make([]personalagent.ConversationSummary, 0, len(summaries))
	for _, summary := range summaries {
		if summary.AssistantMode || summary.Archived || summary.MessageCount == 0 {
			continue
		}
		if query != "" {
			conversation, exists := records[summary.ID]
			if !exists || !conversationMatchesQuery(summary, conversation, query) {
				continue
			}
		}
		filtered = append(filtered, summary)
	}
	return filtered
}

func conversationMatchesQuery(
	summary personalagent.ConversationSummary,
	conversation personalagent.Conversation,
	query string,
) bool {
	if strings.Contains(strings.ToLower(summary.Title+" "+summary.ProjectID), query) {
		return true
	}
	for _, message := range conversation.Messages {
		if strings.Contains(strings.ToLower(message.Content), query) {
			return true
		}
	}
	return false
}

func removeLastRune(value string) string {
	if value == "" {
		return value
	}
	_, size := utf8.DecodeLastRuneInString(value)
	return value[:len(value)-size]
}

func (ui *terminalUI) sendMessage() error {
	content := strings.TrimSpace(ui.composer)
	ui.composer = ""
	ui.mode = tuiConversation
	if content == "" {
		ui.status = "Empty messages are not sent."
		return nil
	}
	if ui.conversation == nil {
		conversation, err := ui.service.CreateConversation(ui.ctx, "", "", "", "", "")
		if err != nil {
			return fmt.Errorf("create conversation: %w", err)
		}
		ui.conversation = &conversation
	}
	turnID, err := ui.service.StartTurn(ui.conversation.ID, content, nil)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	conversation, err := ui.service.Conversation(ui.conversation.ID)
	if err != nil {
		return fmt.Errorf("reload conversation: %w", err)
	}
	ui.conversation = &conversation
	ui.streaming = ""
	ui.status = "Waiting for provider..."
	history, updates, unsubscribe := ui.service.Subscribe(turnID, 0)
	defer unsubscribe()
	for _, event := range history {
		finished, err := ui.applyTurnEvent(turnID, event)
		if err != nil || finished {
			return err
		}
	}
	for {
		select {
		case <-ui.ctx.Done():
			_ = ui.service.CancelTurn(ui.conversation.ID, turnID)
			return nil
		case key, open := <-ui.keys:
			if open && key == 3 {
				if err := ui.service.CancelTurn(ui.conversation.ID, turnID); err != nil {
					return fmt.Errorf("cancel turn: %w", err)
				}
				ui.status = "Cancelling turn..."
			}
		case event, open := <-updates:
			if !open {
				return errors.New("turn event stream closed")
			}
			finished, err := ui.applyTurnEvent(turnID, event)
			if err != nil || finished {
				return err
			}
		}
	}
}

func (ui *terminalUI) applyTurnEvent(turnID string, event personalagent.TurnEvent) (bool, error) {
	switch event.Type {
	case "delta":
		if content, ok := event.Data["content"].(string); ok {
			ui.streaming += content
		}
	case "status":
		if status, ok := event.Data["status"].(string); ok {
			ui.status = status
		}
	case "approval_required":
		return false, ui.confirmApproval(turnID, event.Data)
	case "error":
		ui.status = eventMessage(event.Data)
	case "complete":
		status, _ := event.Data["status"].(string)
		ui.status = "Turn " + status + "."
		ui.streaming = ""
		conversation, err := ui.service.Conversation(ui.conversation.ID)
		if err != nil {
			return true, fmt.Errorf("reload completed conversation: %w", err)
		}
		ui.conversation = &conversation
		return true, nil
	}
	return false, ui.draw()
}

func (ui *terminalUI) confirmApproval(turnID string, data map[string]any) error {
	requestID, _ := data["request_id"].(string)
	if requestID == "" {
		return errors.New("provider requested approval without a request ID")
	}
	encoded, err := json.Marshal(data["params"])
	if err != nil {
		return fmt.Errorf("format approval details: %w", err)
	}
	_, _ = fmt.Fprintf(
		ui.output,
		"\r\nApproval required: %s %s\r\nApprove? [y/N] ",
		sanitizeTerminalText(fmt.Sprint(data["method"])),
		sanitizeTerminalText(string(encoded)),
	)
	var decision string
	for {
		select {
		case <-ui.ctx.Done():
			return ui.ctx.Err()
		case key, open := <-ui.keys:
			if !open {
				return errors.New("terminal input closed during approval")
			}
			switch key {
			case 'y', 'Y':
				decision = "accept"
			case 'n', 'N', '\r', '\n', 3:
				decision = "decline"
			default:
				continue
			}
		}
		break
	}
	if err := ui.service.DecideApproval(ui.conversation.ID, turnID, requestID, decision); err != nil {
		return fmt.Errorf("record approval decision: %w", err)
	}
	ui.status = "Approval " + decision + "."
	return nil
}

func eventMessage(data map[string]any) string {
	message, ok := data["message"].(string)
	if !ok || strings.TrimSpace(message) == "" {
		return "Turn failed."
	}
	return "Turn failed: " + sanitizeTerminalText(message)
}

func (ui *terminalUI) draw() error {
	width, height, err := term.GetSize(int(ui.input.Fd()))
	if err != nil || width < 40 || height < 10 {
		width, height = 80, 24
	}
	_, _ = fmt.Fprint(ui.output, "\x1b[H\x1b[2J\x1b[?25l")
	switch ui.mode {
	case tuiList, tuiSearch:
		_, _ = fmt.Fprintln(ui.output, "Cockpit Agent | j/k move  Enter open  / search  n new  q quit")
		if ui.mode == tuiSearch {
			_, _ = fmt.Fprintf(ui.output, "Search: %s_\n", sanitizeTerminalText(ui.composer))
		} else if ui.search != "" {
			_, _ = fmt.Fprintf(ui.output, "Filter: %s  (press / to change)\n", sanitizeTerminalText(ui.search))
		}
		if len(ui.conversations) == 0 {
			_, _ = fmt.Fprintln(ui.output, "No conversations found. Press n to start a chat.")
		}
		limit := height - 5
		for index, conversation := range ui.conversations {
			if index >= limit {
				break
			}
			marker := "  "
			if index == ui.selected {
				marker = "> "
			}
			line := fmt.Sprintf("%s%s  %s  %d messages",
				marker, sanitizeTerminalText(conversation.Title),
				conversation.UpdatedAt.Local().Format(time.DateTime), conversation.MessageCount)
			_, _ = fmt.Fprintln(ui.output, truncateTerminalText(line, width))
		}
	case tuiConversation, tuiCompose:
		title := "New conversation"
		if ui.conversation != nil {
			title = ui.conversation.Title
		}
		_, _ = fmt.Fprintf(ui.output, "%s | i write  b back  q quit\n\n", truncateTerminalText(sanitizeTerminalText(title), width))
		lines := ui.messageLines(width - 2)
		start := len(lines) - (height - 7)
		if start < 0 {
			start = 0
		}
		for _, line := range lines[start:] {
			_, _ = fmt.Fprintln(ui.output, line)
		}
		if ui.mode == tuiCompose {
			_, _ = fmt.Fprintf(ui.output, "\n> %s_\n", sanitizeTerminalText(ui.composer))
		}
	}
	if ui.status != "" {
		_, _ = fmt.Fprintln(ui.output, truncateTerminalText(sanitizeTerminalText(ui.status), width))
	}
	_, err = fmt.Fprint(ui.output, "\x1b[?25h")
	return err
}

func (ui *terminalUI) messageLines(width int) []string {
	if width < 20 {
		width = 78
	}
	var lines []string
	if ui.conversation != nil {
		for _, message := range ui.conversation.Messages {
			role := "Agent"
			if message.Role == "user" {
				role = "You"
			}
			lines = append(lines, wrapTerminalText(role+": "+message.Content, width)...)
		}
	}
	if ui.streaming != "" {
		lines = append(lines, wrapTerminalText("Agent: "+ui.streaming, width)...)
	}
	if len(lines) == 0 {
		return []string{"No messages yet."}
	}
	return lines
}

func wrapTerminalText(value string, width int) []string {
	value = sanitizeTerminalText(value)
	if width < 1 {
		width = 1
	}
	var lines []string
	for _, paragraph := range strings.Split(value, "\n") {
		runes := []rune(paragraph)
		for len(runes) > width {
			lines = append(lines, string(runes[:width]))
			runes = runes[width:]
		}
		lines = append(lines, string(runes))
	}
	return lines
}

func sanitizeTerminalText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	return strings.Map(func(character rune) rune {
		if character == '\n' || character == '\t' || !unicode.IsControl(character) {
			return character
		}
		return ' '
	}, value)
}

func truncateTerminalText(value string, width int) string {
	runes := []rune(value)
	if width > 0 && len(runes) > width {
		if width < 4 {
			return string(runes[:width])
		}
		return string(runes[:width-3]) + "..."
	}
	return value
}
