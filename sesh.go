package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type CreateSessionRequest struct {
	Name *string
}

type CreateSessionResponse struct {
	Id uuid.UUID
}

type CloseSessionRequest struct {
	Id *uuid.UUID
}

type CloseSessionResponse struct {
	Message string
	Status  uint
}

type Session struct {
	Id           uuid.UUID
	Name         string
	CreationTime string
	Filepath     string
}

type ListSession struct {
	Sessions []Session
}

func CheckError(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}

func main() {
	defaultPath, osError := os.Getwd()
	CheckError(osError)
	logDir := flag.String("log-dir", defaultPath, "Directory to put all log files")

	// Session related channels
	createSessionReq := make(chan CreateSessionRequest)
	createSessionRes := make(chan CreateSessionResponse)
	listSessionReq := make(chan bool)
	listSessionRes := make(chan []Session)
	closeSessionReq := make(chan CloseSessionRequest)
	closeSessionRes := make(chan CloseSessionResponse)

	// Session manager
	go func() {

		sessions := make(map[uuid.UUID]Session)

		for {
			select {
			case createSession := <-createSessionReq:
				id, _ := uuid.NewRandom()
				creationTime := time.Now().Format(time.RFC3339)
				sessions[id] = Session{
					Id:           id,
					Name:         *createSession.Name,
					CreationTime: creationTime,
					Filepath:     filepath.Join(*logDir, fmt.Sprintf("%s-%s-%s", *createSession.Name, creationTime, id.String()[:8])),
				}
				createSessionRes <- CreateSessionResponse{id}
			case <-listSessionReq:
				var results []Session
				for k := range sessions {
					results = append(results, sessions[k])
				}
				listSessionRes <- results
			case closeSession := <-closeSessionReq:
				id := *closeSession.Id
				_, exists := sessions[id]
				if exists {
					delete(sessions, id)
					closeSessionRes <- CloseSessionResponse{fmt.Sprintf("Successfully closed session with id %s\n", id.String()), http.StatusOK}
				} else {
					closeSessionRes <- CloseSessionResponse{fmt.Sprintf("Session id %s does not exist\n", id.String()), http.StatusBadRequest}
				}
			}
		}

	}()

	http.HandleFunc("/create-session", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			decoder := json.NewDecoder(r.Body)
			var newSession CreateSessionRequest
			err := decoder.Decode(&newSession)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if newSession.Name == nil {
				http.Error(w, "Invalid create session object", http.StatusBadRequest)
				return
			}
			createSessionReq <- newSession
			response := <-createSessionRes

			json.NewEncoder(w).Encode(response)
			w.Header().Add("Content-Type", "application/json")
			w.Header().Add("Status", fmt.Sprint(http.StatusOK))
			fmt.Printf("Session created with id=%s\n", response.Id)
		default:
			http.Error(w, "Not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/list-sessions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.Header().Add("Content-Type", "application/json")
			listSessionReq <- true
			sessions := <-listSessionRes
			json.NewEncoder(w).Encode(ListSession{sessions})
			w.Header().Add("Status", fmt.Sprint(http.StatusOK))
		default:
			http.Error(w, "Not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/close-session", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			var closeSession CloseSessionRequest
			json.NewDecoder(r.Body).Decode(&closeSession)
			closeSessionReq <- closeSession
			result := <-closeSessionRes
			w.Header().Add("Status", fmt.Sprint(result.Status))
			fmt.Fprintf(w, result.Message)
		}
	})

	err := http.ListenAndServe(":8080", nil)
	CheckError(err)
}
