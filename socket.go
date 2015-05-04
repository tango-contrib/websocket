package websocket

import (
	"errors"
	"net"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lunny/tango"
)

type Socket struct {
	*Options

	// The websocket Socket
	ws *websocket.Conn

	// The remote Address of the client using this Socket. Cached on the
	// Socket for logging.
	remoteAddr net.Addr

	// The error channel is given the error object as soon as an error occurs
	// either sending or receiving values from the websocket. This channel gets
	// mapped for the next handler to use.
	Error chan error

	// The disconnect channel is for listening for disconnects from the next handler.
	// Any sends to the disconnect channel lead to disconnecting the socket with the
	// given closing message. This channel gets mapped for the next
	// handler to use.
	Disconnect chan int

	// The done channel gets called only when the Socket
	// has been successfully disconnected. Any sends to the disconnect
	// channel are currently ignored. This channel gets mapped for the next
	// handler to use.
	Done chan bool

	// The internal disconnect channel. Sending on this channel will lead to the handlers and
	// the Socket closing.
	disconnect chan error

	// The disconnect send channel. Sending on this channel will lead to the send handler and
	// closing.
	disconnectSend chan bool

	Sender chan []byte

	// the ticker for pinging the client.
	ticker *time.Ticker

	log tango.Logger

	onClosed func(*websocket.Conn)
}

// Creates a new Socket
func newSocket(ws *websocket.Conn, o *Options, logger tango.Logger, onClosed func(*websocket.Conn)) *Socket {
	return &Socket{
		o,
		ws,
		ws.RemoteAddr(),
		make(chan error, 1),
		make(chan int, 1),
		make(chan bool, 3),
		make(chan error, 1),
		make(chan bool, 1),
		make(chan []byte, o.MaxMessageSize),
		nil,
		logger,
		onClosed,
	}
}

// Ping the client through the websocket
func (c *Socket) ping() error {
	c.log.Debug("Pinging socket")
	return c.ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(c.WriteWait))
}

// Start the ticker used for pinging the client
func (c *Socket) startTicker() {
	c.log.Debugf("Pinging every %v, first at %v", c.PingPeriod, time.Now().Add(c.PingPeriod))
	c.ticker = time.NewTicker(c.PingPeriod)
}

// Stop the ticker used for pinging the client
func (c *Socket) stopTicker() {
	c.log.Debug("Stopped pinging socket")
	c.ticker.Stop()
}

func (c *Socket) disconnectChannel() chan error {
	return c.disconnect
}

func (c *Socket) DisconnectChannel() chan int {
	return c.Disconnect
}

func (c *Socket) ErrorChannel() chan error {
	return c.Error
}

// Keep the Socket alive by refreshing the deadlines.
func (c *Socket) keepAlive() {
	c.log.Debugf("Setting read deadline to %v", time.Now().Add(c.PongWait))
	c.ws.SetReadDeadline(time.Now().Add(c.PongWait))
	if c.WriteWait == 0 {
		c.log.Debug("Write deadline set to 0, will never expire")
		c.ws.SetWriteDeadline(time.Time{})
	} else {
		c.log.Debugf("Setting write deadline to %v", time.Now().Add(c.WriteWait))
		c.ws.SetWriteDeadline(time.Now().Add(c.WriteWait))
	}
}

// Set the gorilla websocket handler options according to given options and set a default pong
// handler to keep the Socket alive
func (c *Socket) setSocketOptions() {
	c.ws.SetReadLimit(c.MaxMessageSize)
	c.keepAlive()
	c.ws.SetPongHandler(func(string) error {
		c.log.Debug("Received Pong from Client")
		c.keepAlive()
		return nil
	})
}

// Close the Base Socket. Closes the send Handler and all channels used
// Since all channels are either internal or channels this middleware is sending on.
func (c *Socket) Close(closeCode int) error {
	c.disconnectSend <- true
	//TODO look for a better way to unblock the reader
	c.ws.SetReadDeadline(time.Now())

	// Send close message to the client
	c.log.Debug("Sending close message to client")
	c.ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(closeCode, ""), time.Now().Add(c.WriteWait))

	// If the Socket can not be closed, return the error
	c.log.Debug("Closing websocket Socket")
	if err := c.ws.Close(); err != nil {
		c.log.Errorf("Socket could not be closed: %s", err.Error())
		return err
	}

	// Send disconnect message to the next handler
	c.log.Debug("Sending disconnect to handler")
	c.Done <- true

	// Close disconnect and error channels this Socket was sending on
	close(c.Done)
	close(c.Error)

	if c.onClosed != nil {
		c.onClosed(c.ws)
	}

	return nil
}

func (c *Socket) send() {
	// Start the ticker and defer stopping it and decrementing the
	// wait group counter.
	c.startTicker()
	defer func() {
		c.stopTicker()
		c.log.Debug("Goroutine sending to websocket has been closed")
	}()

	for {
		select {
		// Receiving a message from the next handler
		case message, ok := <-c.Sender:
			if !ok {
				c.log.Error("Sender channel has been closed")
				c.disconnect <- errors.New("Sender channel has been closed")
				return
			}
			// Write the message as a byte array to the socket
			c.log.Debugf("Writing %s to socket", message)

			c.keepAlive()
			if err := c.ws.WriteMessage(websocket.TextMessage, message); err != nil {
				c.log.Errorf("Error writing to socket: %s", err)
				c.disconnect <- err
				return
			}

			c.keepAlive()
		// Ping the client
		case <-c.ticker.C:
			err := c.ping()
			c.log.Debugf("%v", err)
			if err := c.ping(); err != nil {
				c.log.Errorf("Error pinging socket: %v", err)
				c.disconnect <- err
				return
			}

		// Receiving disconnectSend from the closing Connection
		case <-c.disconnectSend:
			return
		}
	}
}

func (c *Socket) recv(onReceived func(*websocket.Conn, []byte)) {
	// Defer decrementing the wait group counter and closing the connection
	defer func() {
		c.log.Debug("Goroutine receiving from websocket has been closed")
	}()

	for {
		// Read a message from the client
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			c.log.Errorf("Error reading from socket: %s", err)
			c.disconnect <- err
			return
		}
		// Send the message as a string to the next handler
		c.log.Debugf("Read message from socket, %s", string(message))

		//c.Receiver <- string(message)
		onReceived(c.ws, message)

		c.keepAlive()
	}
}
