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
	"github.com/gorilla/schema"
)

var (
	ongoingNotifications = make(map[string](map[string]struct{ close chan bool }))
)

type Broker struct {
	Notifier       map[string]chan string
	MasterNotifier chan struct{ Id_Company string }
	NewClients     chan struct {
		Id_Company  string
		Destination string
	}
	ClosingClients chan struct {
		Id_Company  string
		Destination string
	}
	Clients map[string][]chan string
}

func handleNotification(w http.ResponseWriter, r *http.Request) {
	p := new(dto.EmitNotification)

	if err := json.NewDecoder(r.Body).Decode(p); err != nil {
		return
	}

	idNotification := uuid.NewString()

	if _, ok := ongoingNotifications[p.Id_Company]; !ok {
		ongoingNotifications[p.Id_Company] = make(map[string]struct{ close chan bool })
	}

	ongoingNotifications[p.Id_Company][p.Destination] = struct{ close chan bool }{close: make(chan bool)}

	go func(n *dto.EmitNotification, ch struct{ close chan bool }) {
		for {
			select {
			case _ = <-ch.close:
				fmt.Println("closing notification", idNotification, "to", n.Destination, "...")
				return
			case <-time.After(time.Duration(n.Pulse) * time.Second):
				fmt.Println("sending to master", n.Id_Company)
				client.MasterNotifier <- struct{ Id_Company string }{n.Id_Company}
			}
		}
	}(p, ongoingNotifications[p.Id_Company][p.Destination])

	return
}

func handleNotificationAck(w http.ResponseWriter, r *http.Request) {
	ack := new(dto.Ack)

	if err := json.NewDecoder(r.Body).Decode(ack); err != nil {
		return
	}

	ongoingNotifications[ack.Id_Company][ack.Id].close <- true

	return
}

func handleSse(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Client: %v", r.RemoteAddr)

	flusher, ok := w.(http.Flusher)

	if !ok {
		http.Error(w, "streaming unsupported!", http.StatusInternalServerError)
		return
	}
	if err := r.ParseForm(); err != nil {
		log.Fatalln(err)
	}

	query := new(struct {
		Destination string `schema:"destination"`
		Id_Company  string `schema:"id_company"`
	})

	if err := schema.NewDecoder().Decode(query, r.Form); err != nil {
		log.Fatalln(err)
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan string)

	if _, ok := client.Clients[query.Id_Company]; !ok {
		fmt.Println(client.Clients, ok)
		client.Clients[query.Id_Company] = append(client.Clients[query.Id_Company], ch)
	}

	go func() {
		select {
		case <-r.Context().Done():
			fmt.Println("\nclient disconnected")
			client.ClosingClients <- struct {
				Id_Company  string
				Destination string
			}{Id_Company: query.Id_Company, Destination: query.Destination}
			return
		}
	}()

	for {
		fmt.Fprintf(w, "data: %s\n\n", <-ch)
		flusher.Flush()
	}
}

var client = &Broker{
	Notifier:       make(map[string]chan string),
	MasterNotifier: make(chan struct{ Id_Company string }),
	NewClients: make(chan struct {
		Id_Company  string
		Destination string
	}),
	ClosingClients: make(chan struct {
		Id_Company  string
		Destination string
	}),
	Clients: make(map[string][]chan string),
}

func Listen() {
	for {
		select {
		case c := <-client.ClosingClients:
			delete(client.Clients, c.Destination)
			log.Printf("Removed client. %d registered clients", len(client.Clients))
		case c := <-client.MasterNotifier:
			fmt.Println("received at master. to", c.Id_Company)
			fmt.Println("should send to", client.Clients[c.Id_Company], client.MasterNotifier)
			for _, ch := range client.Clients[c.Id_Company] {
				ch <- "message to " + c.Id_Company
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
