package web

import (
	"context"

	"github.com/mrbanja/watchparty-vlc-client/tools/http_client"

	"github.com/bufbuild/connect-go"
	protocol "github.com/mrbanja/watchparty-proto/gen-go"
	protoconnect "github.com/mrbanja/watchparty-proto/gen-go/protocolconnect"
	"github.com/rs/xid"
	"go.uber.org/zap"
)

type Client struct {
	ID string

	client protoconnect.PartyServiceClient
	conn   *connect.BidiStreamForClient[protocol.RoomRequest, protocol.RoomResponse]
	logger *zap.Logger
}

func New(serverAddr string, logger *zap.Logger) *Client {
	hc := http_client.New()
	client := protoconnect.NewPartyServiceClient(hc, serverAddr)
	ID := xid.New().String()
	defer logger.Info("Web Client created", zap.String("ID", ID), zap.String("Addr", serverAddr))
	return &Client{
		ID:     ID,
		client: client,
		logger: logger.Named("Web").With(zap.String("ID", ID)),
	}
}

func (c *Client) MustJoinRoom(ctx context.Context) {
	c.logger.Info("Joined room")
	c.conn = c.client.JoinRoom(ctx)
	c.conn.RequestHeader().Set("X-Client-Id", c.ID)
	err := c.conn.Send(&protocol.RoomRequest{
		Data: &protocol.RoomRequest_Connect{Connect: &protocol.Connect{RoomName: "party"}},
	})
	if err != nil {
		c.logger.Fatal("Error while joining the room", zap.Error(err))
	}
}

func (c *Client) GetMagnet(ctx context.Context) (string, error) {
	resp, err := c.client.GetMagnet(ctx, connect.NewRequest(&protocol.GetMagnetRequest{RoomName: "party"}))
	if err != nil {
		return "", err
	}
	return resp.Msg.Magnet, nil
}

func (c *Client) EnforceLogger(logger *zap.Logger) {
	c.logger = logger.Named("Web")
}

func (c *Client) Listen(ctx context.Context) (<-chan Message, error) {
	if c.conn == nil {
		c.MustJoinRoom(ctx)
	}
	respCh := make(chan Message)
	go func() {
		defer func() {
			_ = c.conn.CloseResponse()
			_ = c.conn.CloseRequest()
		}()
		for {
			msg, err := c.conn.Receive()
			if err != nil {
				c.logger.Error("Error while read. Closing channel, exiting", zap.Error(err))
				close(respCh)
				return
			}
			respCh <- Message{
				Time:   int(msg.Update.Time),
				Status: dto2dao_State(msg.Update.State),
			}
		}
	}()
	return respCh, nil
}

func (c *Client) Send(m Message) error {
	c.logger.Info("Sending message", zap.Any("Message", m))
	return c.conn.Send(&protocol.RoomRequest{Data: &protocol.RoomRequest_Update{Update: &protocol.Update{
		State: dao2dto_State(m.Status),
		Time:  float32(m.Time),
	}}})
}
