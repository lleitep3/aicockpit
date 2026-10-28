package personalagent

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//go:embed web
var webAssets embed.FS

const maximumRequestBodyBytes = 10 << 20

type HTTPHandler struct {
	service   *Service
	token     string
	host      string
	origin    string
	dashboard DashboardOverviewSource
}

func NewHTTPHandler(service *Service, token, host string) http.Handler {
	return NewHTTPHandlerWithDashboard(service, token, host, nil)
}

func NewHTTPHandlerWithDashboard(service *Service, token, host string, dashboard DashboardOverviewSource) http.Handler {
	handler := &HTTPHandler{
		service: service, token: token, host: host, origin: "http://" + host,
		dashboard: dashboard,
	}
	assets, _ := fs.Sub(webAssets, "web")
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.serveIndex)
	mux.Handle("/app.css", http.FileServer(http.FS(assets)))
	mux.Handle("/markdown.js", http.FileServer(http.FS(assets)))
	mux.Handle("/interaction.js", http.FileServer(http.FS(assets)))
	mux.Handle("/preview.js", http.FileServer(http.FS(assets)))
	mux.Handle("/quota.js", http.FileServer(http.FS(assets)))
	mux.Handle("/app.js", http.FileServer(http.FS(assets)))
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets))))
	mux.HandleFunc("/api/v1/", handler.handleAPI)
	return handler.securityHeaders(handler.hostGuard(mux))
}

func (h *HTTPHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := webAssets.ReadFile("web/index.html")
	if err != nil {
		http.Error(w, "Cockpit interface is unavailable", http.StatusInternalServerError)
		return
	}
	page := strings.Replace(string(data), "__COCKPIT_TOKEN__", h.token, 1)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.WriteString(w, page)
}

func (h *HTTPHandler) handleAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if !h.authorized(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Local session token is missing or invalid.")
		return
	}
	if isMutation(r.Method) && !h.validMutationOrigin(r) {
		writeError(w, http.StatusForbidden, "origin_rejected", "Request origin is not allowed.")
		return
	}
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) == 2 && parts[0] == "dashboard" && parts[1] == "overview" && r.Method == http.MethodGet {
		if h.dashboard == nil {
			writeError(w, http.StatusServiceUnavailable, "dashboard_unavailable", "Cockpit Dashboard bridge is unavailable.")
			return
		}
		overview, err := h.dashboard.Overview(r.Context())
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "dashboard_unavailable", "Cockpit Dashboard overview is unavailable.")
			return
		}
		writeJSON(w, http.StatusOK, overview)
		return
	}
	if len(parts) == 1 && parts[0] == "providers" && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{"providers": h.service.Providers()})
		return
	}
	if len(parts) == 3 && parts[0] == "providers" && parts[2] == "models" && r.Method == http.MethodGet {
		catalog, err := h.service.Models(r.Context(), parts[1])
		if err != nil {
			writeError(w, http.StatusConflict, "catalog_unavailable", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, catalog)
		return
	}
	if len(parts) == 1 && parts[0] == "workspace" && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]string{"path": h.service.Workspace()})
		return
	}
	if len(parts) == 1 && parts[0] == "routing" {
		switch r.Method {
		case http.MethodGet:
			config, err := h.service.Routing(r.Context())
			if err != nil {
				writeError(w, http.StatusConflict, "routing_unavailable", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, config)
		case http.MethodPut:
			var config RoutingConfig
			if !decodeBody(w, r, &config) {
				return
			}
			updated, err := h.service.UpdateRouting(r.Context(), config)
			if err != nil {
				writeError(w, http.StatusBadRequest, "routing_update_failed", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, updated)
		default:
			methodNotAllowed(w)
		}
		return
	}
	if len(parts) == 1 && parts[0] == "projects" {
		h.projects(w, r)
		return
	}
	if len(parts) > 0 && parts[0] == "assistant" {
		h.assistant(w, r, parts[1:])
		return
	}
	if len(parts) >= 2 && parts[0] == "projects" {
		h.project(w, r, parts[1], parts[2:])
		return
	}
	if len(parts) == 1 && parts[0] == "bots" {
		switch r.Method {
		case http.MethodGet:
			bots, err := h.service.Bots(r.Context())
			if err != nil {
				writeError(w, http.StatusBadGateway, "bots_list_failed", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, bots)
		case http.MethodPost:
			var profile any
			if !decodeBody(w, r, &profile) {
				return
			}
			bot, err := h.service.CreateBot(r.Context(), profile)
			if err != nil {
				writeError(w, http.StatusBadRequest, "bot_create_failed", err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, bot)
		default:
			methodNotAllowed(w)
		}
		return
	}
	if len(parts) == 2 && parts[0] == "bots" {
		switch r.Method {
		case http.MethodGet:
			bot, err := h.service.GetBot(r.Context(), parts[1])
			if err != nil {
				writeError(w, http.StatusNotFound, "bot_not_found", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, bot)
		case http.MethodPut:
			var profile any
			if !decodeBody(w, r, &profile) {
				return
			}
			bot, err := h.service.UpdateBot(r.Context(), parts[1], profile)
			if err != nil {
				writeError(w, http.StatusBadRequest, "bot_update_failed", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, bot)
		case http.MethodDelete:
			if err := h.service.DeleteBot(r.Context(), parts[1]); err != nil {
				writeError(w, http.StatusBadGateway, "bot_delete_failed", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
		default:
			methodNotAllowed(w)
		}
		return
	}
	if len(parts) == 3 && parts[0] == "bots" && parts[2] == "observers" {
		switch r.Method {
		case http.MethodGet:
			observers, err := h.service.ListObservers(r.Context(), parts[1])
			if err != nil {
				writeError(w, http.StatusBadGateway, "observers_list_failed", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, observers)
		case http.MethodPost:
			var obs any
			if !decodeBody(w, r, &obs) {
				return
			}
			created, err := h.service.CreateObserver(r.Context(), parts[1], obs, nil)
			if err != nil {
				writeError(w, http.StatusBadRequest, "observer_create_failed", err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, created)
		default:
			methodNotAllowed(w)
		}
		return
	}
	if len(parts) == 5 && parts[0] == "bots" && parts[2] == "observers" {
		switch r.Method {
		case http.MethodPut:
			var status any
			if !decodeBody(w, r, &status) {
				return
			}
			updated, err := h.service.UpdateObserver(r.Context(), parts[1], parts[3], status)
			if err != nil {
				writeError(w, http.StatusBadRequest, "observer_update_failed", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, updated)
		case http.MethodDelete:
			if err := h.service.DeleteObserver(r.Context(), parts[1], parts[3]); err != nil {
				writeError(w, http.StatusBadRequest, "observer_delete_failed", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
		default:
			methodNotAllowed(w)
		}
		return
	}
	if len(parts) == 1 && parts[0] == "conversations" {
		h.conversations(w, r)
		return
	}
	if len(parts) >= 2 && parts[0] == "conversations" {
		h.conversation(w, r, parts[1], parts[2:])
		return
	}
	writeError(w, http.StatusNotFound, "not_found", "API route not found.")
}

func (h *HTTPHandler) assistant(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 1 && parts[0] == "providers" && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{"providers": h.service.AssistantProviders()})
		return
	}
	if len(parts) == 1 && parts[0] == "sessions" {
		switch r.Method {
		case http.MethodGet:
			sessions, err := h.service.AssistantSessions()
			if err != nil {
				writeError(w, http.StatusInternalServerError, "storage_error", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
		case http.MethodPost:
			var request struct {
				ModelID         string `json:"model_id"`
				ReasoningEffort string `json:"reasoning_effort"`
			}
			if !decodeBody(w, r, &request) {
				return
			}
			session, err := h.service.CreateAssistantSession(r.Context(), request.ModelID, request.ReasoningEffort)
			if err != nil {
				writeError(w, http.StatusBadRequest, "assistant_session_create_failed", err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, session)
		default:
			methodNotAllowed(w)
		}
		return
	}
	if len(parts) < 2 || parts[0] != "sessions" {
		writeError(w, http.StatusNotFound, "not_found", "Assistant route not found.")
		return
	}
	sessionID := parts[1]
	session, err := h.service.Conversation(sessionID)
	if err != nil || !session.AssistantMode {
		writeError(w, http.StatusNotFound, "assistant_session_not_found", "Assistant session was not found.")
		return
	}
	if len(parts) == 2 && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, session)
		return
	}
	if len(parts) == 3 && parts[2] == "messages" && r.Method == http.MethodPost {
		var request struct {
			Content     string       `json:"content"`
			Attachments []Attachment `json:"attachments"`
		}
		if !decodeBody(w, r, &request) {
			return
		}
		turnID, err := h.service.StartTurn(sessionID, request.Content, request.Attachments)
		if err != nil {
			writeError(w, http.StatusConflict, "assistant_turn_start_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"turn_id": turnID})
		return
	}
	if len(parts) == 5 && parts[2] == "turns" && parts[4] == "events" && r.Method == http.MethodGet {
		h.streamTurn(w, r, sessionID, parts[3])
		return
	}
	if len(parts) == 5 && parts[2] == "turns" && parts[4] == "cancel" && r.Method == http.MethodPost {
		if err := h.service.CancelTurn(sessionID, parts[3]); err != nil {
			writeError(w, http.StatusConflict, "turn_cancel_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "cancelling"})
		return
	}
	if len(parts) == 5 && parts[2] == "actions" && parts[4] == "decision" && r.Method == http.MethodPost {
		var request struct {
			Decision string `json:"decision"`
		}
		if !decodeBody(w, r, &request) {
			return
		}
		if err := h.service.DecideAssistantAction(sessionID, parts[3], request.Decision); err != nil {
			writeError(w, http.StatusConflict, "assistant_action_decision_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": request.Decision})
		return
	}
	writeError(w, http.StatusNotFound, "not_found", "Assistant route not found.")
}

func (h *HTTPHandler) projects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		projects, err := h.service.Projects()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "storage_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"projects": projects})
	case http.MethodPost:
		var input ProjectInput
		if !decodeBody(w, r, &input) {
			return
		}
		project, err := h.service.CreateProject(r.Context(), input)
		if err != nil {
			writeError(w, http.StatusBadRequest, "project_create_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, project)
	default:
		methodNotAllowed(w)
	}
}

func (h *HTTPHandler) project(w http.ResponseWriter, r *http.Request, id string, parts []string) {
	if id == "ai-draft" && len(parts) == 0 && r.Method == http.MethodPost {
		var request struct {
			Brief           string `json:"brief"`
			ProviderID      string `json:"provider_id"`
			ModelID         string `json:"model_id"`
			ReasoningEffort string `json:"reasoning_effort"`
		}
		if !decodeBody(w, r, &request) {
			return
		}
		draft, err := h.service.DraftProjectWithAI(
			r.Context(), request.Brief, request.ProviderID, request.ModelID, request.ReasoningEffort,
		)
		if err != nil {
			writeError(w, http.StatusBadRequest, "project_draft_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, draft)
		return
	}
	if len(parts) == 0 {
		switch r.Method {
		case http.MethodGet:
			project, err := h.service.Project(id)
			if err != nil {
				writeError(w, http.StatusNotFound, "project_not_found", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, project)
		case http.MethodPatch:
			var input ProjectInput
			if !decodeBody(w, r, &input) {
				return
			}
			project, err := h.service.UpdateProject(r.Context(), id, input)
			if err != nil {
				writeError(w, http.StatusBadRequest, "project_update_failed", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, project)
		case http.MethodDelete:
			if err := h.service.DeleteProject(id); err != nil {
				writeError(w, http.StatusNotFound, "project_delete_failed", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		default:
			methodNotAllowed(w)
		}
		return
	}
	writeError(w, http.StatusNotFound, "not_found", "Project route not found.")
}

func (h *HTTPHandler) conversations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if r.URL.Query().Get("summary") == "true" {
			summaries, err := h.service.ConversationSummaries()
			if err != nil {
				writeError(w, http.StatusInternalServerError, "storage_error", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"conversations": summaries})
			return
		}
		conversations, err := h.service.Conversations()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "storage_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"conversations": conversations})
	case http.MethodPost:
		var request struct {
			ProjectID            string `json:"project_id"`
			ParentConversationID string `json:"parent_conversation_id"`
			Title                string `json:"title"`
			ProviderID           string `json:"provider_id"`
			ModelID              string `json:"model_id"`
			ReasoningEffort      string `json:"reasoning_effort"`
			ApprovalPolicy       string `json:"approval_policy"`
			RoutingMode          string `json:"routing_mode"`
		}
		if !decodeBody(w, r, &request) {
			return
		}
		conversation, err := h.service.CreateConversationWithParent(
			r.Context(), request.ProjectID, request.ParentConversationID, request.Title,
			request.ProviderID, request.ModelID, request.ReasoningEffort,
			request.ApprovalPolicy, request.RoutingMode,
		)
		if err != nil {
			writeError(w, http.StatusBadRequest, "conversation_create_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, conversation)
	default:
		methodNotAllowed(w)
	}
}

func (h *HTTPHandler) conversation(w http.ResponseWriter, r *http.Request, id string, parts []string) {
	if len(parts) == 0 && r.Method == http.MethodGet {
		conversation, err := h.service.Conversation(id)
		if err != nil {
			writeError(w, http.StatusNotFound, "conversation_not_found", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, conversation)
		return
	}
	if len(parts) == 1 && parts[0] == "project" && r.Method == http.MethodPatch {
		var request struct {
			ProjectID string `json:"project_id"`
		}
		if !decodeBody(w, r, &request) {
			return
		}
		conversation, err := h.service.AssignConversationProject(id, request.ProjectID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "project_assign_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, conversation)
		return
	}
	if len(parts) == 1 && parts[0] == "archive" && r.Method == http.MethodPatch {
		var request struct {
			Archived bool `json:"archived"`
		}
		if !decodeBody(w, r, &request) {
			return
		}
		conversation, err := h.service.ArchiveConversation(id, request.Archived)
		if err != nil {
			writeError(w, http.StatusBadRequest, "conversation_archive_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, conversation)
		return
	}
	if len(parts) == 1 && parts[0] == "parent" && r.Method == http.MethodPatch {
		var request struct {
			ParentConversationID string `json:"parent_conversation_id"`
		}
		if !decodeBody(w, r, &request) {
			return
		}
		conversation, err := h.service.AssignConversationParent(id, request.ParentConversationID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "parent_assign_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, conversation)
		return
	}
	if len(parts) == 1 && parts[0] == "selection" && r.Method == http.MethodPatch {
		var request struct {
			ProviderID      string `json:"provider_id"`
			ModelID         string `json:"model_id"`
			ReasoningEffort string `json:"reasoning_effort"`
			ApprovalPolicy  string `json:"approval_policy"`
			RoutingMode     string `json:"routing_mode"`
		}
		if !decodeBody(w, r, &request) {
			return
		}
		conversation, err := h.service.UpdateSelection(
			r.Context(), id, request.ProviderID, request.ModelID,
			request.ReasoningEffort, request.ApprovalPolicy, request.RoutingMode,
		)
		if err != nil {
			writeError(w, http.StatusConflict, "selection_update_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, conversation.Selection)
		return
	}
	if len(parts) == 1 && parts[0] == "messages" && r.Method == http.MethodPost {
		var request struct {
			Content     string       `json:"content"`
			Attachments []Attachment `json:"attachments"`
		}
		if !decodeBody(w, r, &request) {
			return
		}
		turnID, err := h.service.StartTurn(id, request.Content, request.Attachments)
		if err != nil {
			writeError(w, http.StatusConflict, "turn_start_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"turn_id": turnID})
		return
	}
	if len(parts) == 3 && parts[0] == "turns" && parts[2] == "events" && r.Method == http.MethodGet {
		h.streamTurn(w, r, id, parts[1])
		return
	}
	if len(parts) == 3 && parts[0] == "turns" && parts[2] == "cancel" && r.Method == http.MethodPost {
		if err := h.service.CancelTurn(id, parts[1]); err != nil {
			writeError(w, http.StatusConflict, "turn_cancel_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "cancelling"})
		return
	}
	if len(parts) == 4 && parts[0] == "turns" && parts[2] == "approvals" && r.Method == http.MethodPost {
		var request struct {
			Decision string `json:"decision"`
		}
		if !decodeBody(w, r, &request) {
			return
		}
		if err := h.service.DecideApproval(id, parts[1], parts[3], request.Decision); err != nil {
			writeError(w, http.StatusConflict, "approval_decision_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": request.Decision})
		return
	}
	writeError(w, http.StatusNotFound, "not_found", "Conversation route not found.")
}

func (h *HTTPHandler) streamTurn(w http.ResponseWriter, r *http.Request, conversationID, turnID string) {
	conversation, err := h.service.Conversation(conversationID)
	if err != nil || !conversationHasTurn(conversation, turnID) {
		writeError(w, http.StatusNotFound, "turn_not_found", "Turn was not found in this conversation.")
		return
	}
	var after uint64
	if value := r.Header.Get("Last-Event-ID"); value != "" {
		after, _ = strconv.ParseUint(value, 10, 64)
	} else if value := r.URL.Query().Get("after"); value != "" {
		after, _ = strconv.ParseUint(value, 10, 64)
	}
	history, updates, unsubscribe := h.service.Subscribe(turnID, after)
	defer unsubscribe()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "stream_unsupported", "Streaming is not supported by this server.")
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()
	for _, event := range history {
		if !writeSSE(w, event) {
			return
		}
		if event.Type == "complete" || event.Type == "error" {
			flusher.Flush()
			return
		}
	}
	for _, turn := range conversation.Turns {
		if turn.ID == turnID && turn.Status != "running" {
			_ = writeSSE(w, TurnEvent{
				ID:         after + uint64(len(history)) + 1,
				Type:       "complete",
				Data:       map[string]any{"status": turn.Status, "error": turn.Error},
				OccurredAt: time.Now().UTC(),
			})
			flusher.Flush()
			return
		}
	}
	flusher.Flush()
	heartbeat := time.NewTicker(10 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case event, open := <-updates:
			if !open {
				return
			}
			if !writeSSE(w, event) {
				return
			}
			flusher.Flush()
			if event.Type == "complete" || event.Type == "error" {
				return
			}
		}
	}
}

func (h *HTTPHandler) authorized(r *http.Request) bool {
	const bearer = "Bearer "
	value := r.Header.Get("Authorization")
	if !strings.HasPrefix(value, bearer) {
		return false
	}
	provided := strings.TrimPrefix(value, bearer)
	return len(provided) == len(h.token) && subtle.ConstantTimeCompare([]byte(provided), []byte(h.token)) == 1
}

func (h *HTTPHandler) validMutationOrigin(r *http.Request) bool {
	if r.Header.Get("Origin") == h.origin {
		return true
	}
	return r.Header.Get("Sec-Fetch-Site") == "same-origin" &&
		r.Header.Get("Sec-Fetch-Mode") == "cors"
}

func (h *HTTPHandler) hostGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != h.host {
			http.Error(w, "invalid host", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *HTTPHandler) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; frame-src http: https:; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func isMutation(method string) bool {
	return method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
}

func decodeBody(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maximumRequestBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_request", "Request body must contain one JSON value.")
		return false
	}
	return true
}

func writeSSE(w io.Writer, event TurnEvent) bool {
	data, err := json.Marshal(event.Data)
	if err != nil {
		return false
	}
	_, err = fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", event.ID, event.Type, data)
	return err == nil
}

func conversationHasTurn(conversation Conversation, turnID string) bool {
	for _, turn := range conversation.Turns {
		if turn.ID == turnID {
			return true
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"code": code, "error": message})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method is not allowed for this route.")
}
