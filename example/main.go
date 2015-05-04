package main

import (
	"sync"
	"time"

	socket "github.com/gorilla/websocket"
	"github.com/lunny/log"
	"github.com/lunny/tango"
	"github.com/tango-contrib/debug"
	"github.com/tango-contrib/renders"
	"github.com/tango-contrib/session"
	"github.com/tango-contrib/websocket"
)

var (
	onlines    = make(map[*socket.Conn]chan []byte)
	onlineLock sync.RWMutex
)

func addOnline(ws *socket.Conn, sender chan []byte) {
	onlineLock.Lock()
	defer onlineLock.Unlock()
	onlines[ws] = sender
}

func sendMsg(ws *socket.Conn, data []byte) {
	onlineLock.RLock()
	defer onlineLock.RUnlock()

	// send to myself
	onlines[ws] <- data

	// send to others
	ticker := time.NewTicker(time.Second * 5)
	for w, sender := range onlines {
		if w != ws {
			select {
			case sender <- data:
			case <-ticker.C:
				goto next
			}
		}
	next:
	}
}

func main() {
	log.Std.SetOutputLevel(log.Lall)
	t := tango.Classic(log.Std)
	t.Use(debug.Debug())
	t.Use(auth())
	t.Use(websocket.New())
	t.Use(session.New())
	t.Use(renders.New(renders.Options{
		Reload: true,
	}))

	t.Get("/", new(Home))
	t.Get("/chat", new(Chat))
	t.Any("/login", new(Login))
	t.Run()
}
