package http_client

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"golang.org/x/net/http2"
)

func New() *http.Client {
	hc := &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, tlsConfig *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
	}
	return hc
}
