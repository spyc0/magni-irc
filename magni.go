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

package magni

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/norwack/magni-irc/irc"
)

// Magni is dark magic at it's best
type Magni struct {
	Nickname string
	Username string
	Realname string
	IRC      *irc.Client
	Proxy    irc.Proxy
	Handlers map[string]func(*Message)
	Channels map[string]bool
}

// Message will hold information about messages and is sent to handlers
type Message struct {
	Nick, Channel, Text string
}

// New will create a new Magni instance
func New(nick, user, real string) *Magni {
	return &Magni{
		Nickname: nick,
		Username: user,
		Realname: real,
		Handlers: make(map[string]func(*Message)),
		Channels: make(map[string]bool),
	}
}

// SetProxy sets the SOCKS5 Proxy to use
func (m *Magni) SetProxy(host string, port int) {
	m.Proxy = irc.Proxy{
		Host: host,
		Port: port,
	}
}

// Handler let's you add handlers for PRIVMSG's
func (m *Magni) Handler(cmd string, h func(*Message)) {
	m.Handlers[cmd] = h
}

// Run will start the IRC connection
func (m *Magni) Run(host string, port int) {
	m.IRC = irc.NewClient()
	m.IRC.Nickname = m.Nickname
	m.IRC.Username = m.Username
	m.IRC.Realname = m.Realname

	err := m.IRC.Connect(host, port, m.Proxy)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for k := range m.Channels {
		m.Send("JOIN " + k)
	}

	for {
		buf := m.IRC.Read()
		cmd := strings.Split(buf, " ")
		if len(cmd) != 4 {
			continue
		}

		if strings.ToLower(cmd[1]) == "privmsg" {
			senderCmd := strings.TrimSpace(cmd[3][1:])
			if _, exists := m.Handlers[senderCmd]; exists {
				senderNick := strings.Split(cmd[0], "!")[0][1:]
				senderChan := cmd[2]
				senderMsg := strings.TrimSpace(cmd[3][1:])
				msg := &Message{
					Nick:    senderNick,
					Channel: senderChan,
					Text:    senderMsg,
				}

				go m.Handlers[senderCmd](msg)
			}
		}
	}
}

// Send sends a message to the server
func (m *Magni) Send(line string) {
	err := m.IRC.Send(line)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

// SendMessage Sends a message to a channel
func (m *Magni) SendMessage(channel string, text string) {
	m.Send(fmt.Sprintf("PRIVMSG %s :%s", channel, text))
}
