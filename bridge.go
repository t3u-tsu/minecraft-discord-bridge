package main

import (
	"regexp"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

// Mojang API response structure
type MojangProfile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func ResolveUUID(username string) (string, error) {
	url := fmt.Sprintf("https://api.mojang.com/users/profiles/minecraft/%s", username)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return "", fmt.Errorf("player not found: %s", username)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("mojang api returned status: %d", resp.StatusCode)
	}

	var profile MojangProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return "", err
	}

	return profile.ID, nil
}

// RCON Packet Types
const (
	RCON_COMMAND      = 2
	RCON_AUTH         = 3
	RCON_RESPONSE     = 2
	RCON_AUTH_RESPONSE = 2
)

type RCONClient struct {
	conn net.Conn
}

// DialRCON: "tcp", "127.0.0.1:25575" or "unix", "/run/minecraft/nitac23s.rcon"
func DialRCON(network, address, password string) (*RCONClient, error) {
	conn, err := net.DialTimeout(network, address, 5*time.Second)
	if err != nil {
		return nil, err
	}

	client := &RCONClient{conn: conn}

	// Authenticate
	if err := client.authenticate(password); err != nil {
		log.Printf("[RCON] Authentication failed for %s: %v", address, err)
		conn.Close()
		return nil, err
	}

	return client, nil
}

func (c *RCONClient) authenticate(password string) error {
	respID, _, err := c.send(RCON_AUTH, password)
	if err != nil {
		return fmt.Errorf("auth send error: %w", err)
	}

	if respID == -1 {
		return errors.New("authentication failed (invalid password)")
	}

	return nil
}

func (c *RCONClient) Execute(command string) (string, error) {
	_, payload, err := c.send(RCON_COMMAND, command)
	if err != nil {
		return "", fmt.Errorf("execute error: %w", err)
	}
	return payload, nil
}

func (c *RCONClient) send(packetType int32, payload string) (int32, string, error) {
	id := int32(time.Now().UnixNano())
	payloadBytes := []byte(payload)
	packetSize := int32(len(payloadBytes) + 10)

	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, packetSize)
	binary.Write(&buf, binary.LittleEndian, id)
	binary.Write(&buf, binary.LittleEndian, packetType)
	buf.Write(payloadBytes)
	buf.Write([]byte{0, 0}) // Null termination for payload and empty string

	if _, err := c.conn.Write(buf.Bytes()); err != nil {
		return 0, "", err
	}

	// Read header (size)
	sizeBuf := make([]byte, 4)
	if _, err := io.ReadFull(c.conn, sizeBuf); err != nil {
		return 0, "", fmt.Errorf("failed to read size: %w", err)
	}

	var resSize int32
	binary.Read(bytes.NewReader(sizeBuf), binary.LittleEndian, &resSize)

	if resSize < 10 || resSize > 4096 {
		return 0, "", fmt.Errorf("invalid response size: %d", resSize)
	}

	// Read rest of the packet
	rest := make([]byte, resSize)
	if _, err := io.ReadFull(c.conn, rest); err != nil {
		return 0, "", fmt.Errorf("failed to read packet body: %w", err)
	}

	var resID, resType int32
	binary.Read(bytes.NewReader(rest[0:4]), binary.LittleEndian, &resID)
	binary.Read(bytes.NewReader(rest[4:8]), binary.LittleEndian, &resType)

	// Payload is from rest[8] to the first null byte
	data := rest[8:]
	nullIdx := bytes.IndexByte(data, 0)
	if nullIdx != -1 {
		data = data[:nullIdx]
	}

	return resID, string(data), nil
}

func (c *RCONClient) Close() {
	c.conn.Close()
}

func GenerateToken(serverName string) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "MC-" + serverName + "-fallback"
	}
	return "MC-" + serverName + "-" + hex.EncodeToString(b)
}
var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\. ]{3,20}$`)

func ValidateMinecraftUsername(username string) bool {
	return usernameRegex.MatchString(username)
}
