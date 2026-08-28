package mpv

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

type Client struct {
	conn net.Conn
	mu   sync.Mutex
}

type Event struct {
	Event  string `json:"event"`
	Reason string `json:"reason,omitempty"`
}

func Connect(socketPath string) (*Client, error) {
	var conn net.Conn
	var err error

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		conn, err = net.Dial("unix", socketPath)
		if err == nil {
			return &Client{conn: conn, mu: sync.Mutex{}}, nil
		}
		time.Sleep(50 * time.Millisecond)
	}

	return nil, fmt.Errorf("connect to mpv: %w", err)
}

func (c *Client) Listen(events chan<- Event) error {
	scanner := bufio.NewScanner(c.conn)

	for scanner.Scan() {
		var event Event
		line := scanner.Bytes()

		if err := json.Unmarshal(line, &event); err != nil {
			return err
		}

		if event.Event != "" {
			events <- event
		}
	}

	return scanner.Err()
}

func (c *Client) Command(command ...any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	msg := map[string]any{
		"command": command,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	data = append(data, '\n')

	_, err = c.conn.Write(data)
	return err
}

func (c *Client) PlaySong(url string) error {
	if err := c.Command("loadfile", url, "replace"); err != nil {
		return err
	}

	return nil
}

func (c *Client) TogglePause() error {
	if err := c.Command("cycle", "pause"); err != nil {
		return err
	}

	return nil
}

func (c *Client) Seek(seconds int) error {
	if err := c.Command("seek", seconds, "relative"); err != nil {
		return err
	}

	return nil
}

func (c *Client) SetVolume(volume int) error {
	if err := c.Command("set_property", "volume", volume); err != nil {
		return err
	}

	return nil
}

func (c *Client) ToggleMute() error {
	if err := c.Command("cycle", "mute"); err != nil {
		return err
	}

	return nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}
