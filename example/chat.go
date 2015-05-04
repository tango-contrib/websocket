package main

import (
	"errors"
	"fmt"

	"github.com/lunny/tango"
	"github.com/tango-contrib/renders"
	"github.com/tango-contrib/websocket"

	socket "github.com/gorilla/websocket"
)

type Home struct {
	Auth
	renders.Renderer
}

func (h *Home) Get() error {
	return h.Render("chat.html", nil)
}

type Chat struct {
	Auth
	tango.Json
	websocket.Message
}

func (c *Chat) Get() error {
	if c.IsUpgrade() {
		return c.OnConnected(func(ws *socket.Conn, sender chan []byte) {
			fmt.Println("on connected")
			addOnline(ws, sender)
		}).OnReceived(func(ws *socket.Conn, data []byte) {
			fmt.Println("on received", string(data))
			sendMsg(ws, data)
		}).OnClosed(func(ws *socket.Conn) {
			fmt.Println("ws closed")
		}).ListenAndServe()
	}

	return errors.New("unsupported protocol")
}
