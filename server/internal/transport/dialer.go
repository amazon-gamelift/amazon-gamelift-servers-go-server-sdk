/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package transport

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"

	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/common"
	"github.com/amazon-gamelift/amazon-gamelift-servers-go-server-sdk/server/log"

	"github.com/gorilla/websocket"
)

// websocketDialer is gorilla/websocket implementation of Dialer interface.
type websocketDialer struct {
	d  *websocket.Dialer
	lg log.ILogger
}

// Dial creates a websocket connection with the specified address.
func (w *websocketDialer) Dial(url string, requestHeader http.Header) (conn Conn, resp *http.Response, err error) {
	if w.lg == nil {
		return w.d.Dial(url, requestHeader)
	}

	ctx := httptrace.WithClientTrace(context.Background(),
		&httptrace.ClientTrace{
			TLSHandshakeStart: func() {
				w.lg.Debugf("TLS handshake starts")
			},
			TLSHandshakeDone: func(state tls.ConnectionState, err error) {
				if err != nil {
					w.lg.Debugf("Error %s when try tls handshake", err)
				} else {
					w.lg.Debugf("TLS handshake OK")
				}
			},
			GotFirstResponseByte: func() {
				w.lg.Debugf("Starts receive response")
			},
			WroteHeaderField: func(key string, value []string) {
				w.lg.Debugf("Request header %s:%s", key, strings.Join(value, ","))
			},
		},
	)
	return w.d.DialContext(ctx, url, requestHeader)
}

// NewDialer create default Dialer instance.
func NewDialer(lg log.ILogger) Dialer {
	return &websocketDialer{
		d: &websocket.Dialer{
			NetDial: func(network, addr string) (net.Conn, error) {
				if lg != nil {
					lg.Debugf("Try connect to the network: %s by addr: %s", network, addr)
				}
				con, err := net.Dial(network, addr)
				if err != nil {
					if lg != nil {
						lg.Debugf("Error when try connect to %s: %s", addr, err)
					}
				} else {
					if lg != nil {
						lg.Debugf("Connection to the %s fine", addr)
					}
				}
				return con, err
			},
			ReadBufferSize:  common.GetEnvIntOrDefault(common.ServiceBufferSize, common.ServiceBufferSizeDefault, nil),
			WriteBufferSize: common.GetEnvIntOrDefault(common.ServiceBufferSize, common.ServiceBufferSizeDefault, nil),
		},
		lg: lg,
	}
}
