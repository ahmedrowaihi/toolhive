package gui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func (s *Server) setupRoutes() {
	api := s.router.PathPrefix("/api").Subrouter()

	api.Use(s.authMiddleware)

	api.HandleFunc("/registry/search", s.handleSearchRegistry).Methods("GET")
	api.HandleFunc("/servers", s.handleListServers).Methods("GET")
	api.HandleFunc("/servers", s.handleRunServer).Methods("POST")
	api.HandleFunc("/servers/{name}/stop", s.handleForceStopServer).Methods("POST")
	api.HandleFunc("/command", s.handleCustomCommand).Methods("POST")

	// static frontend
	s.router.PathPrefix("/").Handler(http.FileServer(http.Dir("pkg/gui/web/static")))
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.service.GetToken() == "" {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != s.service.GetToken() {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleListServers(w http.ResponseWriter, r *http.Request) {
	servers, err := s.service.ListServers(r.Context())
	if err != nil {
		json.NewEncoder(w).Encode(NewErrorResponse(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NewSuccessResponse(servers))
}

func (s *Server) handleRunServer(w http.ResponseWriter, r *http.Request) {
	var request RunServerRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		json.NewEncoder(w).Encode(NewErrorResponse(err))
		return
	}

	if request.Name == "" {
		json.NewEncoder(w).Encode(NewErrorResponse(fmt.Errorf("name is required")))
		return
	}

	output, err := s.service.RunServer(r.Context(), request.Name)
	if err != nil {
		json.NewEncoder(w).Encode(NewErrorResponse(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NewSuccessResponse(output))
}

func (s *Server) handleForceStopServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	if err := s.service.StopServer(r.Context(), name); err != nil {
		json.NewEncoder(w).Encode(NewErrorResponse(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NewSuccessResponse("Server stopped successfully"))
}

func (s *Server) handleCustomCommand(w http.ResponseWriter, r *http.Request) {
	var request CustomCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		json.NewEncoder(w).Encode(NewErrorResponse(err))
		return
	}

	if request.Command == "" {
		json.NewEncoder(w).Encode(NewErrorResponse(fmt.Errorf("command is required")))
		return
	}

	output, err := s.service.RunCommand(r.Context(), request.Command)
	if err != nil {
		json.NewEncoder(w).Encode(NewErrorResponse(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NewSuccessResponse(output))
}

func (s *Server) handleSearchRegistry(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		json.NewEncoder(w).Encode(NewErrorResponse(fmt.Errorf("search query is required")))
		return
	}

	servers, err := s.service.SearchRegistry(query)
	if err != nil {
		json.NewEncoder(w).Encode(NewErrorResponse(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NewSuccessResponse(servers))
}
