package websocket

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lunny/tango"
)

var (
	defaultWriteWait               = 60 * time.Second
	defaultPongWait                = 60 * time.Second
	defaultPingPeriod              = (defaultPongWait * 8 / 10)
	defaultMaxMessageSize    int64 = 65536
	defaultSendChannelBuffer       = 10
	defaultRecvChannelBuffer       = 10
)

type Options struct {
	// The time to wait between writes before timing out the connection
	// When this is a zero value time instance, write will never time out
	WriteWait time.Duration

	// The time to wait at maximum between receiving pings from the client.
	PongWait time.Duration

	// The time to wait between sending pings to the client
	PingPeriod time.Duration

	// The maximum messages size for receiving and sending in bytes
	MaxMessageSize int64

	// The send channel buffer
	SendChannelBuffer int

	// The receiving channel buffer
	RecvChannelBuffer int
}

func prepareOptions(opts []Options) Options {
	var option Options
	if len(opts) > 0 {
		option = opts[0]
	}
	if option.WriteWait == 0 {
		option.WriteWait = defaultWriteWait
	}
	if option.PongWait == 0 {
		option.PongWait = defaultPongWait
	}
	if option.PingPeriod == 0 {
		option.PingPeriod = defaultPingPeriod
	}
	if option.MaxMessageSize == 0 {
		option.MaxMessageSize = defaultMaxMessageSize
	}
	if option.SendChannelBuffer == 0 {
		option.SendChannelBuffer = defaultSendChannelBuffer
	}
	if option.RecvChannelBuffer == 0 {
		option.RecvChannelBuffer = defaultRecvChannelBuffer
	}
	return option
}

func isUpgradeRequest(req *http.Request) bool {
	if req.Method != "GET" {
		return false
	}

	if r, err := regexp.MatchString("https?://"+req.Host+"$", req.Header.Get("Origin")); !r || err != nil {
		return false
	}

	return true
}

// Upgrade the connection to a websocket connection
func upgradeRequest(resp http.ResponseWriter, req *http.Request, o *Options, log tango.Logger) (*websocket.Conn, int, error) {
	if req.Method != "GET" {
		log.Warnf("Remote %s Method %s is not allowed", req.RemoteAddr, req.Method)
		return nil, http.StatusMethodNotAllowed, errors.New("Method not allowed")
	}

	if r, err := regexp.MatchString("https?://"+req.Host+"$", req.Header.Get("Origin")); !r || err != nil {
		log.Warnf("Remote %s Origin %s is not allowed", req.RemoteAddr, req.Host)
		return nil, http.StatusForbidden, errors.New("Origin not allowed")
	}

	log.Debugf("Remote %s Request to %s has been allowed for origin %s", req.RemoteAddr, req.Host, req.Header.Get("Origin"))

	ws, err := websocket.Upgrade(resp, req, nil, 1024, 1024)
	if handshakeErr, ok := err.(websocket.HandshakeError); ok {
		log.Warnf("Remote %s Handshake failed: %s", req.RemoteAddr, handshakeErr)
		return nil, http.StatusBadRequest, handshakeErr
	} else if err != nil {
		log.Warnf("Remote %s Handshake failed: %s", req.RemoteAddr, err)
		return nil, http.StatusBadRequest, err
	}

	log.Infof("Remote %s Connection established", req.RemoteAddr)
	return ws, http.StatusOK, nil
}

// Waits for a disconnect message and closes the connection with an appropriate close message.
// The possible messages are:
// TODO this should get more elaborate.
// CloseNormalClosure           = 1000
// CloseGoingAway               = 1001
// CloseProtocolError           = 1002
// CloseUnsupportedData         = 1003
// CloseNoStatusReceived        = 1005
// CloseAbnormalClosure         = 1006
// CloseInvalidFramePayloadData = 1007
// ClosePolicyViolation         = 1008
// CloseMessageTooBig           = 1009
// CloseMandatoryExtension      = 1010
// CloseInternalServerErr       = 1011
// CloseTLSHandshake            = 1015
func waitForDisconnect(c *Socket) {
	for {
		select {
		case err := <-c.disconnectChannel():
			if err == io.EOF {
				c.ErrorChannel() <- err
				c.Close(websocket.CloseNormalClosure)
			} else {
				c.Close(websocket.CloseAbnormalClosure)
			}

			return
		case closeCode := <-c.DisconnectChannel():
			c.Close(closeCode)
			return
		}
	}
}

func New(options ...Options) tango.HandlerFunc {
	opt := prepareOptions(options)
	return func(ctx *tango.Context) {
		// if not upgrade connetct, ignore
		if !isUpgradeRequest(ctx.Req()) {
			ctx.Next()
			return
		}

		if c, ok := ctx.Action().(message); ok {
			fmt.Println("ssssss")
			c.Init(&opt, ctx)
		} else {
			panic("")
		}
		ctx.Next()
	}
}
