// Copyright 2015-2016 trivago GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package consumer

import (
	"fmt"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"gopkg.in/mcuadros/go-syslog.v2"
	"gopkg.in/mcuadros/go-syslog.v2/format"
	"sync"
)

// Syslogd consumer plugin
// The syslogd consumer accepts messages from a syslogd comaptible socket.
// When attached to a fuse, this consumer will stop the syslogd service in case
// that fuse is burned.
// Configuration example
//
//  - "consumer.Syslogd":
//    Address: "udp://0.0.0.0:514"
//    Format: "RFC6587"
//
// Address defines the protocol, host and port or socket to bind to.
// This can either be any ip address and port like "localhost:5880" or a file
// like "unix:///var/gollum.socket". By default this is set to "udp://0.0.0.0:514".
// The protocol can be defined along with the address, e.g. "tcp://..." but
// this may be ignored if a certain protocol format does not support the desired
// transport protocol.
//
// Format defines the syslog standard to expect for message encoding.
// Three standards are currently supported, by default this is set to "RFC6587".
//  * RFC3164 (https://tools.ietf.org/html/rfc3164) udp only.
//  * RFC5424 (https://tools.ietf.org/html/rfc5424) udp only.
//  * RFC6587 (https://tools.ietf.org/html/rfc6587) tcp or udp.
type Syslogd struct {
	core.ConsumerBase
	format   format.Format // RFC3164, RFC5424 or RFC6587?
	protocol string
	address  string
	sequence *uint64
}

func init() {
	shared.TypeRegistry.Register(Syslogd{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Syslogd) Configure(conf core.PluginConfig) error {
	err := cons.ConsumerBase.Configure(conf)
	if err != nil {
		return err
	}

	cons.address, cons.protocol = shared.ParseAddress(conf.GetString("Address", "udp://0.0.0.0:514"))
	format := conf.GetString("Format", "RFC6587")

	switch cons.protocol {
	case "udp", "tcp", "unix":
	default:
		return fmt.Errorf("Syslog: unknown protocol type %s", cons.protocol) // ### return, unknown protocol ###
	}

	switch format {
	// http://www.ietf.org/rfc/rfc3164.txt
	case "RFC3164":
		cons.format = syslog.RFC3164
		if cons.protocol == "tcp" {
			Log.Warning.Print("Syslog: RFC3164 demands UDP")
			cons.protocol = "udp"
		}

	// https://tools.ietf.org/html/rfc5424
	case "RFC5424":
		cons.format = syslog.RFC5424
		if cons.protocol == "tcp" {
			Log.Warning.Print("Syslog: RFC5424 demands UDP")
			cons.protocol = "udp"
		}

	// https://tools.ietf.org/html/rfc6587
	case "RFC6587":
		cons.format = syslog.RFC6587

	default:
		err = fmt.Errorf("Syslog: Format %s is not supported", format)
	}

	cons.sequence = new(uint64)
	return err
}

// Handle implements the syslog handle interface
func (cons *Syslogd) Handle(parts format.LogParts, code int64, err error) {
	content := ""
	isString := false

	switch cons.format {
	case syslog.RFC3164:
		content, isString = parts["content"].(string)
	case syslog.RFC5424, syslog.RFC6587:
		content, isString = parts["message"].(string)
	default:
		Log.Error.Print("Could not determine the format to retrieve message/content")
	}

	if !isString {
		Log.Error.Print("Message/Content is not a string")
		return
	}

	cons.Enqueue([]byte(content), *cons.sequence)
	*cons.sequence++
}

// Consume opens a new syslog socket.
// Messages are expected to be separated by \n.
func (cons *Syslogd) Consume(workers *sync.WaitGroup) {
	server := syslog.NewServer()
	server.SetFormat(cons.format)
	server.SetHandler(cons)

	switch cons.protocol {
	case "unix":
		if err := server.ListenUnixgram(cons.address); err != nil {
			Log.Error.Print("Syslog: Failed to open unix://", cons.address)
		}
	case "udp":
		if err := server.ListenUDP(cons.address); err != nil {
			Log.Error.Print("Syslog: Failed to open udp://", cons.address)
		}
	case "tcp":
		if err := server.ListenTCP(cons.address); err != nil {
			Log.Error.Print("Syslog: Failed to open tcp://", cons.address)
		}
	}

	server.Boot()
	defer server.Kill()

	cons.SetFuseBurnedCallback(func() { server.Kill() })
	cons.SetFuseActiveCallback(func() { server.Boot() })
	cons.ControlLoop()

	server.Wait()
}
