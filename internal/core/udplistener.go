package core

import (
	"encoding/json"
	"log"
	"net"

	"huginn/internal/types"
)

func RunUDPListener(addr string, reg *Registry) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Printf("core: UDP heartbeat listener on %s", addr)

	buf := make([]byte, 4096)
	for {
		n, from, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("core: udp read error: %v", err)
			continue
		}
		var hb types.Heartbeat
		if err := json.Unmarshal(buf[:n], &hb); err != nil {
			log.Printf("core: dropping malformed heartbeat from %s: %v", from, err)
			continue
		}
		if hb.InstanceID == "" {
			log.Printf("core: dropping heartbeat with empty instance_id from %s", from)
			continue
		}

		reg.Heartbeat(hb.InstanceID, hb.PlayerCount)
	}
}
