package gcm

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/agl/xmpp"
)

const (
	ProductionAddr = "gcm.googleapis.com:5235"
	TestingAddr    = "gcm-preprod.googleapis.com:5236"
)

type Conn struct {
	c    *xmpp.Conn
	rawc net.Conn
	err  error
}

func Dial(addr, senderID, apiKey string) (*Conn, error) {
	con, err := tls.Dial("tcp", addr, nil)
	if err != nil {
		return nil, err
	}

	conf := &xmpp.Config{Conn: con, SkipTLS: true}

	xcon, err := xmpp.Dial(addr, senderID, "gcm.googleapis.com", apiKey, conf)
	if err != nil {
		return nil, err
	}
	xcon.SetCustomStorage(xmpp.NsClient, "message", message{})
	return &Conn{c: xcon, rawc: con}, nil
}

func (c *Conn) Send(m Message) error {
	jsm, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return c.c.SendStanza(wrapMsg(jsm))
}

func (c *Conn) Responses() <-chan Response {
	rch := make(chan Response)
	go func() {
		defer close(rch)
		for {
			s, err := c.c.Next()
			if err != nil {
				c.err = err
				return
			}
			m := s.Value.(*message)
			if m.Error.Code != "" || m.Error.Text.Body != "" {
				c.err = m.Error
				return
			}
			resp := Response{}
			if err = json.Unmarshal(m.GCM.Value, &resp); err != nil {
				c.err = err
				return
			}
			rch <- resp
		}
	}()
	return rch
}

func (c *Conn) SetWriteTimeout(d time.Duration) error {
	return c.rawc.SetWriteDeadline(time.Now().Add(d))
}

func (c *Conn) Close() error {
	return c.rawc.Close()
}

func (c *Conn) Err() error {
	return c.err
}

type Message struct {
	To                       string      `json:"to"`
	ID                       string      `json:"message_id"`
	Type                     string      `json:"message_type,omitempty"`
	CollapseKey              string      `json:"collapse_key,omitempty"`
	Data                     interface{} `json:"data"`
	DelayWhileIdle           bool        `json:"delay_while_idle,omitempty"`
	TTL                      int         `json:"time_to_live,omitempty"`
	DeliveryReceiptRequested bool        `json:"delivery_receipt_requeste,omitempty"`
}

type Response struct {
	Type             string `json:"message_type"`
	ID               string `json:"message_id"`
	From             string `json:"from"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type message struct {
	ID    string     `xml:"id,attr"`
	GCM   gcmWrapper `xml:"gcm"`
	Error gcmError   `xml:"error"`
}

type gcmError struct {
	Code string       `xml:"code,attr"`
	Text gcmErrorText `xml:"text"`
}

func (err gcmError) Error() string {
	return fmt.Sprintf("%s: %s", err.Code, err.Text.Body)
}

type gcmErrorText struct {
	Body string `xml:",innerxml"`
}

type gcmWrapper struct {
	XMLNS string `xml:"xmlns,attr"`
	Value []byte `xml:",innerxml"`
}

func wrapMsg(m []byte) interface{} {
	return message{
		GCM: gcmWrapper{
			XMLNS: "google:mobile:data",
			Value: m,
		},
	}
}
