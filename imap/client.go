package imap

import (
	"errors"
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
	"io"
	"io/ioutil"
	"log"

	"jaytaylor.com/html2text"
)

type MailMetaData struct {
}

type Client struct {
	c             *client.Client
	mailCount     int
	serverAddress string
	email         string
	password      string
}

func NewClient(serverAddress string, email string, password string) *Client {
	c := &Client{}
	c.serverAddress = serverAddress
	c.email = email
	c.password = password
	return c
}

func (C *Client) Login() error {
	c, err := client.DialTLS(C.serverAddress, nil)
	if err != nil {
		return err
	}
	log.Println("Connected")

	// Login
	if err := c.Login(C.email, C.password); err != nil {
		return err
	}
	log.Println("Logged in")
	C.c = c
	return nil
}

func (C *Client) Logout() error {
	return C.c.Logout()
}

func (C *Client) PullMailCount() (int, error) {
	// Select INBOX
	mbox, err := C.c.Select("INBOX", false)
	if err != nil {
		return 0, err
	}
	return int(mbox.Messages), nil
}

func (C *Client) GetMailCount() int {
	return C.mailCount
}

func (C *Client) UpdateMailCount() (int, error) {
	t, err := C.PullMailCount()
	if err != nil {
		return 0, err
	}
	C.mailCount = t
	return t, nil
}

/* Check new mails
this method will update mailCount
return a range [l,r]
return [0,0] if there's no new mail
*/
func (C *Client) GetNewMailRanges() (int, int, error) {
	l := C.mailCount
	r, err := C.UpdateMailCount()
	if err != nil {
		return 0, 0, err
	}
	if l >= r {
		return 0, 0, nil
	}
	return l + 1, r, nil
}

/*  read a mail, return a formatted text and meta-data
TODO: meta-data
*/
func (C *Client) ReadMail(id int) (string, MailMetaData, error) {
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uint32(id))

	// Get the whole message body
	var section imap.BodySectionName
	items := []imap.FetchItem{section.FetchItem()}

	messages := make(chan *imap.Message, 1)
	if err := C.c.Fetch(seqSet, items, messages); err != nil {
		return "", MailMetaData{}, err
	}

	msg := <-messages
	text, err := readMessage(msg, section)
	if err != nil {
		return "", MailMetaData{}, err
	} else {
		return text, MailMetaData{}, nil
	}
}

func readMessage(msg *imap.Message, section imap.BodySectionName) (text string, err error) {
	if msg == nil {
		return "", errors.New("server didn't returned message")
	}

	r := msg.GetBody(&section)
	if r == nil {
		return "", errors.New("server didn't returned message body")
	}

	// Create a new mail reader
	mr, err := mail.CreateReader(r)
	if err != nil {
		return "", err
	}

	// Print some info about the message
	header := mr.Header
	if date, err := header.Date(); err == nil {
		text += fmt.Sprintln("Date:", date)
	}
	if from, err := header.AddressList("From"); err == nil {
		text += "From: "
		for _, i := range from {
			text += fmt.Sprint(*i)
			text += ", "
		}
		if len(text) > 400 {
			text = text[0:400] + "......"
		}
		text += "\n"

	}
	if to, err := header.AddressList("To"); err == nil {
		text += "To: "
		for _, i := range to {
			text += fmt.Sprint(*i)
			text += ", "
		}
		if len(text) > 400 {
			text = text[0:400] + "......"
		}
		text += "\n"
	}
	if subject, err := header.Subject(); err == nil {
		text += fmt.Sprintln("Subject: ", subject)
	}

	flag := true
	// Process each message's part
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			if flag {
				flag = false
			} else {
				continue
			}
			// This is the message's text (can be plain-text or HTML)
			b, _ := ioutil.ReadAll(p.Body)
			txt, err := html2text.FromString(string(b), html2text.Options{})
			if err != nil {
				txt = "<--debug message: html parser error, fallback to source text.-->\n" + string(b)
			}
			text += fmt.Sprintf("Body: %v", txt)
		case *mail.AttachmentHeader:
			// This is an attachment
			filename, _ := h.Filename()
			text += fmt.Sprintf("Got attachment: %v", filename)
		}
		text += "\n"
	}
	return
}
