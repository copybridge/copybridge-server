package server

import (
	"encoding/json"
	"klipx-server/internal/clipboard"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/", s.HelloWorldHandler)

	r.Get("/health", s.healthHandler)

	r.Get("/clipboard/{name}", s.GetHandler)
	r.Post("/clipboard", s.PostHandler)
	r.Put("/clipboard/{name}", s.PutHandler)
	r.Delete("/clipboard/{name}", s.DeleteHandler)

	return r
}

func (s *Server) HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	jsonResp, _ := json.Marshal(s.db.Health())
	_, _ = w.Write(jsonResp)
}

func (s *Server) GetHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	c, err := s.db.Get(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if c == nil {
		http.Error(w, "clipboard not found", http.StatusNotFound)
		return
	}

	if c.IsEncrypted {
		_, password, ok := r.BasicAuth()
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if !c.Authenticate(password) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if err := c.Decrypt(password); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	jsonResp, _ := json.Marshal(c)
	_, _ = w.Write(jsonResp)
}

func (s *Server) PostHandler(w http.ResponseWriter, r *http.Request) {
	var cNew clipboard.Clipboard
	if err := json.NewDecoder(r.Body).Decode(&cNew); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Received clipboard: %+v", cNew)

	c, err := s.db.Get(cNew.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if c != nil {
		http.Error(w, "clipboard already exists", http.StatusConflict)
		return
	}

	if cNew.IsEncrypted {
		_, password, ok := r.BasicAuth()
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		cNew.PasswordHash, err = clipboard.HashPassword(password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cNew.Encrypt(password)
	}

	log.Printf("Processed clipboard: %+v", cNew)

	if err := s.db.Insert(&cNew); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResp, _ := json.Marshal(cNew)
	_, _ = w.Write(jsonResp)
}

func (s *Server) PutHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	c, err := s.db.Get(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if c == nil {
		http.Error(w, "clipboard not found", http.StatusNotFound)
		return
	}

	var cNew clipboard.Clipboard
	if err := json.NewDecoder(r.Body).Decode(&cNew); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	c.DataType = cNew.DataType
	c.Data = cNew.Data

	log.Printf("Received clipboard: %+v", cNew)

	if c.IsEncrypted {
		_, password, ok := r.BasicAuth()
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if !c.Authenticate(password) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		c.Encrypt(password)
	}

	log.Printf("Processed clipboard: %+v", c)

	if err := s.db.Update(c); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResp, _ := json.Marshal(c)
	_, _ = w.Write(jsonResp)
}

func (s *Server) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	c, err := s.db.Get(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if c == nil {
		http.Error(w, "clipboard not found", http.StatusNotFound)
		return
	}

	if c.IsEncrypted {
		_, password, ok := r.BasicAuth()
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if !c.Authenticate(password) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	if err := s.db.Delete(name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
