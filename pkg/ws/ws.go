package ws

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/rs/xid"

	"github.com/fasthttp/websocket"
	"go.uber.org/zap"
)

type Status string

const (
	Play  Status = "play"
	Pause Status = "pause"
)

type Message struct {
	Time   int
	Status Status
}

type Client struct {
	ID     string
	addr   string
	conn   *websocket.Conn
	logger *zap.Logger
}

func New(serverAddr string, logger *zap.Logger) *Client {
	ID := xid.New().String()
	defer logger.Info("WS client created", zap.String("ID", ID))
	return &Client{
		ID:     ID,
		addr:   fmt.Sprintf("ws://%s/ws/party", serverAddr),
		logger: logger.Named("WS").With(zap.String("ID", ID)),
	}
}

func (c *Client) EnforceLogger(logger *zap.Logger) {
	c.logger = logger.Named("WS").With(zap.String("ID", c.ID))
}

func (c *Client) MustConnect(ctx context.Context) {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, c.addr, http.Header{"X-Client-Id": []string{c.ID}})
	if err != nil {
		c.logger.Panic("Error while connect", zap.Error(err))
	}
	c.conn = conn
}

func (c *Client) Listen(ctx context.Context) (<-chan Message, error) {
	if c.conn == nil {
		c.MustConnect(ctx)
	}

	respCh := make(chan Message)
	go func() {
		defer func() {
			_ = c.conn.Close()
		}()
		for {
			_, msg, err := c.conn.ReadMessage()
			if err != nil {
				c.logger.Error("Error while read. Closing channel, exiting", zap.Error(err))
				close(respCh)
				return
			}
			respRow := strings.Split(string(msg), ";")
			time, err := strconv.ParseFloat(respRow[1], 32)
			if err != nil {
				c.logger.Error("Error while parse time", zap.Error(err))
				close(respCh)
				return
			}
			respCh <- Message{
				Time:   int(time),
				Status: Status(respRow[0]),
			}
		}
	}()
	return respCh, nil
}

func (c *Client) Send(m Message) error {
	c.logger.Info("Sending message", zap.Any("Message", m))
	msg := []byte(fmt.Sprintf("%s;%s", m.Status, strconv.Itoa(m.Time)))
	return c.conn.WriteMessage(websocket.TextMessage, msg)
}
