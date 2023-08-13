package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type CreateSessionOp struct {
	Name *string
}

type Session struct {
	Id uuid.UUID
	CreateSessionOp
	CreationTime string
	Filepath     string
}

type ListSession struct {
	Sessions []Session
}

// Perform command line or environment checks
func CanRun() bool {
	return true
}

func main() {

	if !CanRun() {
		log.Fatal("Can't run because environment variables haven't been set")
	}

	// Session related channels
	createSessionOps := make(chan CreateSessionOp, 1)
	listSessionReq := make(chan bool)
	listSessionResp := make(chan []Session)

	// Session manager
	go func() {

		sessions := make(map[uuid.UUID]Session)

		for {
			select {
			case createSession := <-createSessionOps:
				id, _ := uuid.NewUUID()
				sessions[id] = Session{
					Id:              id,
					CreateSessionOp: createSession,
					CreationTime:    time.Now().Format(time.RFC3339),
					Filepath:        "some/path",
				}
			case <-listSessionReq:
				var results []Session
				for k := range sessions {
					results = append(results, sessions[k])
				}
				listSessionResp <- results
			}
		}

	}()

	http.HandleFunc("/create-session", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			decoder := json.NewDecoder(r.Body)
			var newSession CreateSessionOp
			err := decoder.Decode(&newSession)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if newSession.Name == nil {
				http.Error(w, "Invalid create session object", http.StatusBadRequest)
				return
			}
			createSessionOps <- newSession
		default:
			http.Error(w, "Not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/list-sessions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.Header().Add("Content-Type", "application/json")
			listSessionReq <- true
			sessions := <-listSessionResp
			json.NewEncoder(w).Encode(ListSession{sessions})
			w.Header().Add("Status", "200")
		default:
			http.Error(w, "Not allowed", http.StatusMethodNotAllowed)
		}
	})

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err.Error())
	}
}
