package websocket

import (
	"github.com/gorilla/websocket"
	"github.com/lunny/tango"
)

type message interface {
	Init(*Options, *tango.Context)
}

var _ message = &Message{}

type Message struct {
	opt         *Options
	ctx         *tango.Context
	onConnected func(*websocket.Conn, chan []byte)
	onReceived  func(*websocket.Conn, []byte)
	onClosed    func(*websocket.Conn)
}

func (m *Message) Init(opt *Options, ctx *tango.Context) {
	m.opt = opt
	m.ctx = ctx
	m.onConnected = func(*websocket.Conn, chan []byte) {}
	m.onReceived = func(*websocket.Conn, []byte) {}
	m.onClosed = func(*websocket.Conn) {}
}

func (m *Message) IsUpgrade() bool {
	return isUpgradeRequest(m.ctx.Req())
}

func (m *Message) OnConnected(onConnected func(*websocket.Conn, chan []byte)) *Message {
	m.onConnected = onConnected
	return m
}

// when received message from client
func (m *Message) OnReceived(onReceived func(*websocket.Conn, []byte)) *Message {
	m.onReceived = onReceived
	return m
}

func (m *Message) OnClosed(onClosed func(*websocket.Conn)) *Message {
	m.onClosed = onClosed
	return m
}

func (m *Message) ListenAndServe() error {
	// Upgrade the request to a websocket connection
	ws, _, err := upgradeRequest(m.ctx.ResponseWriter, m.ctx.Req(), m.opt, m.ctx.Logger)
	if err != nil {
		return err
	}

	conn := newSocket(ws, m.opt, m.ctx.Logger, m.onClosed)

	m.onConnected(ws, conn.Sender)

	// Set the options for the gorilla websocket package
	conn.setSocketOptions()

	// start the send and receive goroutines
	go conn.send()
	go conn.recv(m.onReceived)
	go waitForDisconnect(conn)
	return nil
}
