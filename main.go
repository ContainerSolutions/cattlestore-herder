// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/gambol99/go-marathon"
	"github.com/gorilla/websocket"
	"path/filepath"
)

const (
	// Time allowed to write the file to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Poll file for changes with this period.
	filePeriod = 10 * time.Second

	marathonAddr = "http://172.17.0.1:8080"
)

var (
	client   marathon.Marathon
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

func reader(ws *websocket.Conn) {
	defer ws.Close()
	ws.SetReadLimit(512)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}

func writer(ws *websocket.Conn) {
	pingTicker := time.NewTicker(pingPeriod)
	fileTicker := time.NewTicker(filePeriod)
	defer func() {
		pingTicker.Stop()
		fileTicker.Stop()
		ws.Close()
	}()
	for {
		select {
		case <-fileTicker.C:
			var p []byte = []byte("hoi")

			// todo get tasks here, jsonise

			if p != nil {
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				if err := ws.WriteMessage(websocket.TextMessage, p); err != nil {
					return
				}
			}
		case <-pingTicker.C:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}

	go writer(ws)
	reader(ws)
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	homeTempl := template.Must(template.ParseFiles(filepath.Join("./", "index.html")))

	log.Printf("%s", client)

	if r.URL.Path != "/" {
		http.Error(w, "Not found", 404)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var v = struct {
		Host string
		Data string
	}{
		r.Host,
		string("initial data goes here"),
	}
	homeTempl.Execute(w, &v)
}

func initMarathonClient() (marathon.Marathon, error) {
	// Configure client
	config := marathon.NewDefaultConfig()
	//	log.Printf(marathonAddr)
	config.URL = "http://172.17.0.1:8080"

	return marathon.NewClient(config)
}

func initMarathon() (marathon.Marathon, marathon.EventsChannel) {
	app, _ := client.Application("cattlestore")
	for _, task := range app.Tasks {
		log.Printf("%s", task.Ports)
	}

	// Register for events
	events := make(marathon.EventsChannel, 25)
	err := client.AddEventsListener(events, marathon.EVENTS_APPLICATIONS)
	if err != nil {
		// todo retry instead of fail
		log.Fatalf("Failed to register for events, %s", err)
	}

	return client, events
}

func main() {
	var err error
	if client, err = initMarathonClient(); err != nil {
		log.Fatalf("No marathon client; %s", err)
	}

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", serveWs)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
