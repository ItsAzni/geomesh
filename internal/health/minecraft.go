package health

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/itsazni/geomesh/internal/config"
)

// CheckMinecraft performs a Minecraft Server List Ping health check.
// Implements the Minecraft 1.7+ ping protocol (modern):
// https://wiki.vg/Server_List_Ping
func CheckMinecraft(ctx context.Context, ep config.EndpointConfig) bool {
	hc := ep.HealthCheck
	port := hc.Port
	if port == 0 {
		port = 25565
	}
	addr := fmt.Sprintf("%s:%d", ep.Address, port)

	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}

	handshake := buildMCHandshake(ep.Address, uint16(port))
	if _, err := conn.Write(handshake); err != nil {
		return false
	}

	statusReq := []byte{0x01, 0x00}
	if _, err := conn.Write(statusReq); err != nil {
		return false
	}

	_, err = readMCVarInt(conn)
	return err == nil
}

// buildMCHandshake constructs a Minecraft Handshake packet.
// Protocol reference: https://wiki.vg/Protocol#Handshake
func buildMCHandshake(host string, port uint16) []byte {
	var body []byte
	body = mcAppendVarInt(body, 0x00)
	body = mcAppendVarInt(body, 764)
	body = mcAppendString(body, host)
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, port)
	body = append(body, portBytes...)
	body = mcAppendVarInt(body, 1)

	var packet []byte
	packet = mcAppendVarInt(packet, len(body))
	packet = append(packet, body...)
	return packet
}

// readMCVarInt reads a single VarInt from a Minecraft connection.
func readMCVarInt(r io.Reader) (int, error) {
	var result int
	var shift uint
	buf := make([]byte, 1)
	for {
		if _, err := io.ReadFull(r, buf); err != nil {
			return 0, err
		}
		b := buf[0]
		result |= int(b&0x7F) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
		if shift >= 35 {
			return 0, fmt.Errorf("VarInt too large")
		}
	}
	return result, nil
}

func mcAppendVarInt(buf []byte, v int) []byte {
	for {
		b := byte(v & 0x7F)
		v >>= 7
		if v != 0 {
			b |= 0x80
		}
		buf = append(buf, b)
		if v == 0 {
			break
		}
	}
	return buf
}

func mcAppendString(buf []byte, s string) []byte {
	buf = mcAppendVarInt(buf, len(s))
	return append(buf, []byte(s)...)
}

func CheckBedrock(ctx context.Context, ep config.EndpointConfig) bool {
	hc := ep.HealthCheck
	port := hc.Port
	if port == 0 {
		port = 19132
	}
	addr := fmt.Sprintf("%s:%d", ep.Address, port)

	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "udp", addr)
	if err != nil {
		return false
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}

	var buf bytes.Buffer
	buf.WriteByte(0x01)

	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBytes, uint64(time.Now().UnixMilli()))
	buf.Write(timeBytes)

	magic := []byte{
		0x00, 0xff, 0xff, 0x00, 0xfe, 0xfe, 0xfe, 0xfe,
		0xfd, 0xfd, 0xfd, 0xfd, 0x12, 0x34, 0x56, 0x78,
	}
	buf.Write(magic)

	guid := make([]byte, 8)
	buf.Write(guid)

	if _, err := conn.Write(buf.Bytes()); err != nil {
		return false
	}

	respBuf := make([]byte, 1024)
	n, err := conn.Read(respBuf)
	if err != nil || n == 0 {
		return false
	}

	if respBuf[0] != 0x1C {
		return false
	}

	return true
}
