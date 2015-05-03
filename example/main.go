package main

import (
	"github.com/lunny/tango"
	"github.com/tango-contrib/renders"
	"github.com/tango-contrib/websocket"
)

func main() {
	t := tango.Classic()
	t.Use(renders.New())
	t.Use(websocket.New())
	t.Use(auth())
	t.Get("/", new(Home))
	t.Get("/static/(*name)", tango.Dir("./static"))
	t.Get("/chat", new(Chat))
	t.Any("/login", new(Login))
	t.Run()
}
