package main

import (
	"github.com/lunny/tango"
	"github.com/tango-contrib/renders"
)

type Auth struct {
}

func auth() tango.HandlerFunc {
	return func(ctx *tango.Context) {
		ctx.Next()
	}
}

type Login struct {
	renders.Renderer
}

func (l *Login) Get() error {
	return l.Render("login.html", nil)
}

func (l *Login) Post() error {
	return nil
}
