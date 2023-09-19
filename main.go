package main

import (
	"clsyan/notifications-go/dto"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
)

var (
	ongoingNotifications = make(map[string](map[string]map[string]struct{ close chan bool }))
)

type Broker struct {
	Notifier       map[string]chan string
	MasterNotifier chan struct {
		IdCompany      string
		IdNotification string
		Payload        map[string]string
	}
	NewClients chan struct {
		IdCompany string
		IdClient  string
	}
	ClosingClients chan struct {
		IdCompany string
		IdClient  string
	}
	Clients map[string]map[string]chan []byte
}

func handleNotification(w http.ResponseWriter, r *http.Request) {
	p := new(dto.EmitNotification)

	if err := json.NewDecoder(r.Body).Decode(p); err != nil {
		return
	}

	if _, ok := ongoingNotifications[p.IdCompany]; !ok {
		ongoingNotifications[p.IdCompany] = make(map[string]map[string]struct{ close chan bool })
	}
	if p.ExpiresInSeconds == 0 {
		p.ExpiresInSeconds = 30
	}
	idNotification := uuid.NewString()
	for clientId := range client.Clients[p.IdCompany] {
		nchan := struct{ close chan bool }{close: make(chan bool)}
		if _, ok := ongoingNotifications[p.IdCompany][clientId]; !ok {
			ongoingNotifications[p.IdCompany][clientId] = make(map[string]struct{ close chan bool })
		}
		ongoingNotifications[p.IdCompany][clientId][idNotification] = nchan

		go func(n *dto.EmitNotification, clientId string, ch struct{ close chan bool }) {
			timeout := time.After(time.Duration(n.ExpiresInSeconds) * time.Second)

			for {
				select {
				case <-ch.close:
					fmt.Println("closing notification", idNotification, "to", clientId, "...")
					return
				case <-timeout:
					fmt.Println("notification", idNotification, "to", clientId, "expired. closing...")
					return
				case <-time.After(time.Duration(n.Pulse) * time.Second):
					fmt.Println("sending to client", clientId, ". total clients", len(client.Clients[p.IdCompany]))
					out, err := json.Marshal(struct {
						IdNotification string
						ClientId       string
						Payload        map[string]string
					}{IdNotification: idNotification, ClientId: clientId, Payload: p.Payload})
					if err != nil {
						log.Fatalln(err)
					}

					client.Clients[p.IdCompany][clientId] <- out
				}
			}
		}(p, clientId, nchan)
	}
}

func handleNotificationAck(w http.ResponseWriter, r *http.Request) {
	ack := new(dto.Ack)

	if err := json.NewDecoder(r.Body).Decode(ack); err != nil {
		return
	}

	ongoingNotifications[ack.IdCompany][ack.IdClient][ack.IdNotification].close <- true
}

func handleSse(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Client: %v\n", r.RemoteAddr)

	flusher, ok := w.(http.Flusher)

	if !ok {
		http.Error(w, "streaming unsupported!", http.StatusInternalServerError)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query := new(struct {
		IdClient  string `schema:"id_client"`
		IdCompany string `schema:"id_company"`
	})

	if err := schema.NewDecoder().Decode(query, r.Form); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !(query.IdClient != "" && query.IdCompany != "") {
		http.Error(w, "id_client and id_company not provided", http.StatusBadRequest)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	if _, ok := client.Clients[query.IdCompany]; !ok {
		client.Clients[query.IdCompany] = make(map[string]chan []byte)
		fmt.Println("client hub for company registered")
	}

	ch := make(chan []byte)
	client.Clients[query.IdCompany][query.IdClient] = ch
	fmt.Println("client", query.IdClient, "registered in hub", query.IdCompany, "total", len(client.Clients[query.IdCompany]))

	for {
		fmt.Fprintf(w, "data: %s\n\n", <-ch)
		flusher.Flush()
	}
}

var client = &Broker{
	MasterNotifier: make(chan struct {
		IdCompany      string
		IdNotification string
		Payload        map[string]string
	}),
	Notifier: make(map[string]chan string),
	NewClients: make(chan struct {
		IdCompany string
		IdClient  string
	}),
	ClosingClients: make(chan struct {
		IdCompany string
		IdClient  string
	}),
	Clients: make(map[string]map[string]chan []byte),
}

func Listen() {
	c := <-client.ClosingClients
	delete(client.Clients[c.IdCompany], c.IdClient)

	for _, cchan := range ongoingNotifications[c.IdCompany][c.IdClient] {
		cchan.close <- true
	}
	log.Printf("Removed client %s. %d registered clients", c.IdClient, len(client.Clients))
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
