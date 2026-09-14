package sidecar

import (
	"encoding/json"
	"net"

	"huginn/internal/types"
)

type UDPSender struct {
	conn *net.UDPConn
}

func NewUDPSender(addr string) (*UDPSender, error) {
	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return nil, err
	}

	return &UDPSender{conn: conn}, nil
}
func (s *UDPSender) Send(hb types.Heartbeat) error {
	data, err := json.Marshal(hb)
	if err != nil {
		return err
	}
	_, err = s.conn.Write(data)
	return err
}
func (s *UDPSender) Close() error {
	return s.conn.Close()
}
