package main

import (
	"clsyan/notifications-go/dto"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

var (
	ongoingNotifications = make(map[string](chan bool))
)

type Broker struct {
	/*
	   Events are pushed to this channel by the main events-gathering routine
	*/
	Notifier chan string

	/*
		New client connections
	*/
	NewClients chan chan string

	/*
		Closed client connections
	*/
	ClosingClients chan chan string

	/*
		Client connections registry
	*/
	Clients map[chan string]bool
}

func handleNotification(w http.ResponseWriter, r *http.Request) {
	p := new(dto.Notification)

	if err := json.NewDecoder(r.Body).Decode(p); err != nil {
		return
	}

	id := uuid.NewString()

	ongoingNotifications[id] = make(chan bool)

	go func(n *dto.Notification, ch chan bool) {
		for {
			select {
			case _ = <-ch:
				fmt.Println("closing notification", "...")
				return
			case <-time.After(time.Duration(n.Pulse) * time.Second):
				fmt.Println("sending notification", id, "to", n.Destination, "...")
				client.Notifier <- n.Destination
			}
		}
	}(p, ongoingNotifications[id])

	return
}

func handleNotificationAck(w http.ResponseWriter, r *http.Request) {
	ack := new(dto.Ack)

	if err := json.NewDecoder(r.Body).Decode(ack); err != nil {
		return
	}

	ongoingNotifications[ack.Id] <- true

	return
}

func handleSse(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Client: %v", r.RemoteAddr)

	flusher, ok := w.(http.Flusher)

	if !ok {
		http.Error(w, "streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	messageChan := make(chan string)

	client.NewClients <- messageChan

	go func() {
		select {
		case <-r.Context().Done():
			fmt.Println("\nclient disconnected")
			client.ClosingClients <- messageChan
			return
		}
	}()

	for {
		fmt.Fprintf(w, "data: %s\n\n", <-client.Notifier)
		flusher.Flush()
	}
}

var client = &Broker{
	Notifier:       make(chan string),
	NewClients:     make(chan chan string),
	ClosingClients: make(chan chan string),
	Clients:        map[chan string]bool{},
}

func Listen() {
	for {
		select {
		case s := <-client.NewClients:
			client.Clients[s] = true
			log.Printf("Client added. %d registered clients", len(client.Clients))
		case s := <-client.ClosingClients:
			delete(client.Clients, s)
			log.Printf("Removed client. %d registered clients", len(client.Clients))
		case event := <-client.Notifier:
			for clientMessageChan := range client.Clients {
				clientMessageChan <- event
			}
		}
	}
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/notifications", handleNotification).Methods("POST")
	r.HandleFunc("/notifications", handleNotificationAck).Methods("PATCH")
	r.HandleFunc("/sse", handleSse).Methods("GET")
	http.Handle("/", r)
	go Listen()
	log.Println("listening at", 3000)
	http.ListenAndServe(":3000", nil)
}
