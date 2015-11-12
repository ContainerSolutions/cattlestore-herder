// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	marathon "github.com/gambol99/go-marathon"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"text/template"
	"time"
)

const (
	// Time allowed to write data to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

var (
	homeTempl = template.Must(template.New("").Parse(homeHTML))
	upgrader  = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	events marathon.EventsChannel
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

func handleEvent(ws *websocket.Conn, event *marathon.Event) error {
	log.Printf("----- %s", event.Event)
	ws.SetWriteDeadline(time.Now().Add(writeWait))
	return ws.WriteMessage(websocket.TextMessage, []byte(event.Name))
}

func writer(ws *websocket.Conn) {
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		pingTicker.Stop()
		ws.Close()
	}()

	log.Printf("%s", events)

	for {
		select {
		case event := <-events:
			log.Printf("+++++ Received event: %s", event)
			if err := handleEvent(ws, event); err != nil {
				return
			}
		case <-pingTicker.C:
			log.Printf("pinging %s", ws.RemoteAddr())
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
		Host    string
		Data    string
		LastMod string
	}{
		r.Host,
		string("website goes here"),
		"1",
	}

	homeTempl.Execute(w, &v)
}

func initMarathon() (marathon.Marathon, marathon.EventsChannel) {
	// Configure client
	config := marathon.NewDefaultConfig()
	config.URL = "http://172.17.0.1:8080"

	client, err := marathon.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create a client for marathon, error: %s", err)
	}

	// Register for events
	events := make(marathon.EventsChannel, 25)
	err = client.AddEventsListener(events, marathon.EVENTS_APPLICATIONS)
	if err != nil {
		// todo retry instead of fail
		log.Fatalf("Failed to register for events, %s", err)
	}

	return client, events
}

func main() {
	log.Print("Starting...")

	client, events := initMarathon()
	defer client.RemoveEventsListener(events)
	log.Printf("%s", events)

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", serveWs)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

const homeHTML = `<!DOCTYPE html>
<html lang="en">
    <head>
        <title>WebSocket Example</title>
    </head>
    <body>
        <pre id="fileData">{{.Data}}</pre>
        <script type="text/javascript">
            (function() {
                var data = document.getElementById("fileData");
                var conn = new WebSocket("ws://{{.Host}}/ws?lastMod={{.LastMod}}");
                conn.onclose = function(evt) {
                    data.textContent = 'Connection closed';
                }
                conn.onmessage = function(evt) {
                    console.log('file updated', evt.data);
                    data.textContent = evt.data;
                }
            })();
        </script>
    </body>
</html>
`
