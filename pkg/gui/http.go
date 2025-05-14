package gui

import (
	"net/http"

	"github.com/gorilla/mux"
)

type Server struct {
	router  *mux.Router
	service *Service
}

func NewServer() *Server {
	s := &Server{
		router:  mux.NewRouter(),
		service: NewService(),
	}
	s.setupRoutes()
	return s
}

func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}
