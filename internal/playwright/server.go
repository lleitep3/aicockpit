package playwright

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type ActionRequest struct {
	Action   string `json:"action"` // goto, click, type, eval
	Selector string `json:"selector,omitempty"`
	Text     string `json:"text,omitempty"`
	URL      string `json:"url,omitempty"`
	JS       string `json:"js,omitempty"`
}

type ActionResponse struct {
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Result  interface{} `json:"result,omitempty"`
}

type Server struct {
	driver *Driver
	mux    *http.ServeMux
	server *http.Server
}

func NewServer(driver *Driver) *Server {
	mux := http.NewServeMux()
	s := &Server{
		driver: driver,
		mux:    mux,
		server: &http.Server{
			Addr:    "127.0.0.1:9091",
			Handler: mux,
		},
	}
	mux.HandleFunc("/action", s.handleAction)
	return s
}

func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

func (s *Server) handleAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respond(w, false, err.Error(), nil)
		return
	}

	var err error
	var result interface{}

	switch req.Action {
	case "goto":
		err = s.driver.Goto(req.URL)
	case "click":
		err = s.driver.Click(req.Selector)
	case "type":
		err = s.driver.Type(req.Selector, req.Text)
	case "eval":
		result, err = s.driver.Eval(req.JS)
	default:
		err = fmt.Errorf("unknown action: %s", req.Action)
	}

	if err != nil {
		s.respond(w, false, err.Error(), nil)
	} else {
		s.respond(w, true, "", result)
	}
}

func (s *Server) respond(w http.ResponseWriter, success bool, errMsg string, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ActionResponse{
		Success: success,
		Error:   errMsg,
		Result:  result,
	})
}
