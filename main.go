// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"github.com/gambol99/go-marathon"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"text/template"
	"time"
)

const (
	// Time allowed to write the file to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// How long to wait between data pushes to the browser.
	dataPushPeriod = 1 * time.Second

	// How long to wait before we can scale again, in order to not overwhelm Marathon.
	scaleCoolDownPeriod time.Duration = 5 * time.Second

	marathonAddr = "http://172.17.0.1:8080"
)

type ClusterState struct {
	Instances     []Instance `json:"instances"`
	NrOfInstances int        `json:"nrOfInstances"`
}

type Instance struct {
	Id  string `json:"id"`
	Max int    `json:"max"`
	Ops int    `json:"ops"`
}
type State struct {
	Max int `json:"max"`
	Ops int `json:"ops"`
}

// custom template delimiters since the Go default delimiters clash with Angular's default.
var templateDelimiters = []string{"{{%", "%}}"}

var lastScaleAction time.Time = time.Unix(0, 0)

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

func scale_up(ops_t float64, max_t float64) {
	load := ops_t / max_t
	reservoir := int(max_t - ops_t)
	log.Printf("load: %f, reservoir: %d, max: %d", load, reservoir, int(max_t))

	if load > 0.5 && reservoir < 120 {
		var diff time.Duration
		diff = time.Now().Sub(lastScaleAction)
		if diff < scaleCoolDownPeriod {
			log.Print("Not scaling due to cooldown period")
			return
		}

		app, err := client.Application("cattlestore")
		if err != nil {
			log.Printf("Not scaling: %s", err)
		}
		newInstances := 2 + app.Instances

		lastScaleAction = time.Now()
		log.Print("SCALE UP DUDE!!!")
		if _, err := client.ScaleApplicationInstances("cattlestore", newInstances, false); err != nil {
			log.Printf("Could not scale: %s", err)
		}
	}
}

func writer(ws *websocket.Conn) {
	pingTicker := time.NewTicker(pingPeriod)
	fileTicker := time.NewTicker(dataPushPeriod)
	defer func() {
		pingTicker.Stop()
		fileTicker.Stop()
		ws.Close()
	}()

	timeout := time.Duration(1 * time.Second)
	httpClient := http.Client{
		Timeout: timeout,
	}

	for {
		select {
		case <-fileTicker.C:
			var clusterState ClusterState

			app, err := client.Application("cattlestore")
			if err != nil {
				log.Printf("Not scaling: %s", err)
			}
			clusterState.NrOfInstances = app.Instances

			ops_t, max_t := 0.0, 0.0

			if app != nil && err == nil {
				for _, task := range app.Tasks {
					resp, err := httpClient.Get(fmt.Sprintf("http://172.17.0.1:%d/info", task.Ports[0]))
					if err != nil {
						continue
					}

					state, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						continue
					}

					f := State{}
					json.Unmarshal(state, &f)
					clusterState.Instances = append(clusterState.Instances, Instance{
						Id:  task.ID[12:20],
						Ops: f.Ops,
						Max: f.Max,
					})
					ops_t += float64(f.Ops)
					max_t += float64(f.Max)
				}
			}

			go scale_up(ops_t, max_t)

			p, err := json.Marshal(clusterState)
			if err != nil {
				continue
			}
			//			p := []byte(`[{"id":"6cd8a58f","max":24,"ops":24},{"id":"6cd8a58e","max":24,"ops":18},
			//			{"id":"1cd8a58f","max":24,"ops":24},{"id":"4cd8a58e","max":24,"ops":18},
			//			{"id":"2cd8a58f","max":24,"ops":24},{"id":"5cd8a58e","max":24,"ops":18},
			//			{"id":"3cd8a58f","max":24,"ops":24},{"id":"7cd8a58e","max":24,"ops":18}, {"id":"6cdf3444","max":30,"ops":12},{"id":"6cbdca8b","max":13,"ops":3},{"id":"6ce8f854","max":13,"ops":12}]`)

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
	temp := template.New("index.html")
	temp.Delims(templateDelimiters[0], templateDelimiters[1])
	temp.ParseFiles(filepath.Join("./", "index.html"))

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
	temp.Execute(w, &v)
}

func initMarathonClient() (marathon.Marathon, error) {
	config := marathon.NewDefaultConfig()
	config.URL = "http://172.17.0.1:8080"
	return marathon.NewClient(config)
}

func main() {
	var err error
	if client, err = initMarathonClient(); err != nil {
		log.Fatalf("No marathon client; %s", err)
	}
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", serveWs)
	log.Print("Ready to herd...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
