package proxy

import (
	"bufio"
	"context"
	"encoding/binary"
	"net"

	"github.com/pion/dtls/v2"
	"github.com/pion/dtls/v2/pkg/protocol"
	"github.com/pion/dtls/v2/pkg/protocol/extension"
	"github.com/pion/dtls/v2/pkg/protocol/handshake"
	"github.com/pion/dtls/v2/pkg/protocol/recordlayer"
	"github.com/pion/udp"
)

// ListenDTLS implements listener for dtls
func ListenDTLS(network string, addr *net.UDPAddr) (net.Listener, error) {
	lc := udp.ListenConfig{
		AcceptFilter: func(packet []byte) bool {
			pkts, err := recordlayer.UnpackDatagram(packet)
			if err != nil || len(pkts) < 1 {
				return false
			}
			h := &recordlayer.Header{}
			if err := h.Unmarshal(pkts[0]); err != nil {
				return false
			}
			return h.ContentType == protocol.ContentTypeHandshake
		},
	}
	return lc.Listen(network, addr)
}

type serveDTLS struct {
	config func(sni string) (*dtls.Config, error)
	target Target
}

func (m serveDTLS) HandleConn(ctx context.Context, conn net.Conn) {
	defer func() {
		conn.Close()
	}()
	if m.config == nil {
		return
	}
	var sni string
	if m := GetContextMeta(ctx); m != nil {
		sni = m.ServerName.Load()
	}
	conf, err := m.config(sni)
	if err != nil {
		return
	}
	newConn, err := dtls.Server(conn, conf)
	if err != nil {
		return
	}
	m.target.HandleConn(ctx, newConn)
}

func unpackPacket(br *bufio.Reader) []byte {
	buf, err := br.Peek(recordlayer.HeaderSize)
	if err != nil {
		return nil
	}
	pktLen := (recordlayer.HeaderSize + int(binary.BigEndian.Uint16(buf[11:])))
	buf, err = br.Peek(pktLen)
	if err != nil {
		return nil
	}
	return buf
}

func clientHelloServerNameDTLS(br *bufio.Reader) string {
	x := unpackPacket(br)
	if x != nil {
		h := &recordlayer.RecordLayer{}
		if err := h.Unmarshal(x); err != nil {
			return ""
		}
		if s, ok := h.Content.(*handshake.Handshake); ok {
			if ch, ok := s.Message.(*handshake.MessageClientHello); ok {
				for _, e := range ch.Extensions {
					if et, ok := e.(*extension.ServerName); ok {
						return et.ServerName
					}
				}
			}
		}
	}
	return ""
}
