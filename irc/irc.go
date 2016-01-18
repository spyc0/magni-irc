/*

The MIT License (MIT)

Copyright (c) 2016 Norwack

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

*/

package irc

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// Proxy is a type that holds information about a SOCKS5 proxy server
type Proxy struct {
	Host string
	Port int
}

// Client type contains information about the client
type Client struct {
	Nickname string
	Username string
	Realname string

	Connected bool

	Conn   net.Conn
	Buffer chan string
}

// NewClient returns a pointer to a new instance of Client
func NewClient() *Client {
	return &Client{
		Buffer: make(chan string),
	}
}

// Connect will connect to the IRC server specified in host and port
func (c *Client) Connect(host string, port int, socksProxy Proxy) error {
	if socksProxy.Host == "" && socksProxy.Port == 0 {
		conn, err := net.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err != nil {
			return err
		}
		c.Conn = conn
	} else {
		dialer, err := proxy.SOCKS5("tcp", net.JoinHostPort(socksProxy.Host, strconv.Itoa(socksProxy.Port)), nil, proxy.Direct)
		conn, err := dialer.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err != nil {
			return err
		}
		c.Conn = conn
	}

	err := c.Send(fmt.Sprintf("NICK %s\r\nUSER %s 0 0 :%s\r\n", c.Nickname, c.Username, c.Realname))
	if err != nil {
		c.Conn.Close()
		return err
	}

	for {
		if c.Connected {
			break
		}
		buf := make([]byte, 1024)
		_, err := c.Conn.Read(buf)
		if err != nil {
			c.Conn.Close()
			return err
		}

		for _, line := range strings.Split(string(buf), "\n") {
			if len(line) == 0 {
				continue
			}
			cmd := strings.Split(line, " ")[0]
			if strings.ToLower(cmd) == "error" {
				c.Connected = false
				c.Conn.Close()
				return fmt.Errorf("server sent error: %s", line)
			}
			if strings.ToLower(cmd) == "ping" {
				c.Send(fmt.Sprintf("PONG %s", strings.Split(line, " ")[1]))
			}
			if len(strings.Split(line, " ")) >= 2 {
				if strings.Split(line, " ")[1] == "376" || strings.Split(line, " ")[1] == "422" {
					c.Connected = true
				}
			}
		}
	}

	time.Sleep(1 * time.Second)

	go c.reader()

	return nil
}

func (c *Client) Read() string {
	return <-c.Buffer
}

// Send can be used to write to the IRC server
func (c *Client) Send(line string) error {
	_, err := c.Conn.Write([]byte(line + "\r\n"))
	if err != nil {
		c.Connected = false
		c.Conn.Close()
		return err
	}
	return nil
}

func (c *Client) reader() {
	for {
		if !c.Connected {
			break
		}

		buf := make([]byte, 1024)
		n, err := c.Conn.Read(buf)
		if err != nil {
			c.Buffer <- "QUIT: Failed to read from server"
			c.Connected = false
			break
		}
		for _, line := range strings.Split(string(buf[:n]), "\r\n") {
			lineLower := strings.ToLower(line)
			if strings.Split(lineLower, " ")[0] == "error" {
				c.Connected = false
			} else if strings.Split(lineLower, " ")[0] == "ping" {
				c.Send(fmt.Sprintf("PONG %s", strings.Split(line, " ")[1]))
			}
			c.Buffer <- line
		}
		time.Sleep(100 * time.Millisecond)
	}
}
