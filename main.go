package main

import (
	"encoding/json"
	"fmt"
	"github.com/WAAutoMaton/telegram-mail-bot/imap"
	"github.com/WAAutoMaton/telegram-mail-bot/smtp"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

type Config struct {
	Token      string
	IMAPServer string
	SMTPServer string
	SMTPHost   string
	Username   string
	Password   string
	UID        int
}

func initBotAPI(token string) (bot *tb.Bot) {
	b, err := tb.NewBot(tb.Settings{
		// You can also set custom API URL.
		// If field is empty it equals to "https://api.telegram.org".
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatal(err)
		return
	}
	return b
}
func readConfig() *Config {
	b, err := ioutil.ReadFile("config/config.json")
	if err != nil {
		panic(err)
	}
	config := &Config{}
	err = json.Unmarshal(b, config)
	if err != nil {
		panic(err)
	}
	return config
}

type DraftMail struct {
	isActive bool
	to       string
	subject  string
	body     string
}

func main() {
	config := readConfig()
	b := initBotAPI(config.Token)

	c := imap.NewClient(config.IMAPServer, config.Username, config.Password)
	err := c.Login()
	if err != nil {
		log.Fatal(err)
	}

	smtpClient := smtp.NewClient(config.SMTPServer, config.SMTPHost, config.Username, config.Password)

	b.Handle("/ping", func(m *tb.Message) {
		if !m.Private() {
			return
		}
		_, err := b.Send(m.Sender, "Pong!")
		logError(err)
	})

	b.Handle("/help", func(m *tb.Message) {
		if !m.Private() {
			return
		}
		if m.Sender.ID != config.UID {
			b.Send(m.Sender, "You aren't my admin!")
			return
		}
		_, err := b.Send(m.Sender, `Command List:
/help    show this message.
/inbox   show the number of mails in INBOX.
/mail <ID:int>   get a mail (ID 1 is the latest mail.)
/new   create a new draft mail.`)
		logError(err)
	})

	b.Handle("/inbox", func(m *tb.Message) {
		if !m.Private() {
			return
		}
		if m.Sender.ID != config.UID {
			b.Send(m.Sender, "You aren't my admin!")
			return
		}
		t, err := c.PullMailCount()
		text := fmt.Sprintf("There are %v mails in INBOX", strconv.Itoa(t))
		if err != nil {
			text = fmt.Sprintf("Catched an error while trying to pull INBOX, error message: %v", err.Error())
		}
		_, err = b.Send(m.Sender, text)
		logError(err)
	})

	b.Handle("/mail", func(m *tb.Message) {
		if !m.Private() {
			return
		}
		if m.Sender.ID != config.UID {
			b.Send(m.Sender, "You aren't my admin!")
			return
		}
		var x int
		_, err := fmt.Sscanf(m.Payload, "%v", &x)
		if err != nil {
			b.Send(m.Sender, "Use /mail <ID:int> to get a mail ( ID 1 is the latest mail )")
			return
		}
		n, err := c.PullMailCount()
		if err != nil {
			b.Send(m.Sender, "Catched an error, error message: "+err.Error())
			return
		}
		text, _, err := c.ReadMail(n - x + 1)
		if err != nil {
			b.Send(m.Sender, "Catched an error, error message: "+err.Error())
			return
		}
		_, err = b.Send(m.Sender, omitText(text))
		logError(err)
	})

	draftMail := DraftMail{isActive: false}
	b.Handle("/new", func(m *tb.Message) {
		if !m.Private() {
			return
		}
		if m.Sender.ID != config.UID {
			b.Send(m.Sender, "You aren't my admin!")
			return
		}
		if draftMail.isActive {
			_, err := b.Send(m.Sender, "Please use /cancel to delete last draft mail first.")
			logError(err)
			return
		}
		draftMail = DraftMail{isActive: true}
		_, err := b.Send(m.Sender,
			`A new draft mail has been created.
Plase use:
/to <receiver's mail :string>
/subject <string>
/body <string>
to fill the mail.
And then use /send to send it.
Or use /cancel to delete the draft mail.`)
		logError(err)
	})
	b.Handle("/cancel", func(m *tb.Message) {
		if !m.Private() {
			return
		}
		if m.Sender.ID != config.UID {
			b.Send(m.Sender, "You aren't my admin!")
			return
		}
		var err error
		if draftMail.isActive {
			draftMail.isActive = false
			_, err = b.Send(m.Sender, "Delete draft mail successfully.")
		} else {
			_, err = b.Send(m.Sender, "No draft mail should be deleted.")
		}
		logError(err)
	})

	b.Handle("/to", func(m *tb.Message) {
		if !m.Private() {
			return
		}
		if m.Sender.ID != config.UID {
			b.Send(m.Sender, "You aren't my admin!")
			return
		}
		if !draftMail.isActive {
			_, err := b.Send(m.Sender, "No draft mail.\nPlease use /new to create one.")
			logError(err)
			return
		}
		to := strings.Trim(m.Payload, " \r\n\t")
		draftMail.to = to
		_, err := b.Send(m.Sender, "Succeed.")
		logError(err)
	})

	b.Handle("/subject", func(m *tb.Message) {
		if !m.Private() {
			return
		}
		if m.Sender.ID != config.UID {
			b.Send(m.Sender, "You aren't my admin!")
			return
		}
		if !draftMail.isActive {
			_, err := b.Send(m.Sender, "No draft mail.\nPlease use /new to create one.")
			logError(err)
			return
		}
		subject := strings.Trim(m.Payload, " \r\n\t")
		draftMail.subject = subject
		_, err := b.Send(m.Sender, "Succeed.")
		logError(err)
	})

	b.Handle("/body", func(m *tb.Message) {
		if !m.Private() {
			return
		}
		if m.Sender.ID != config.UID {
			b.Send(m.Sender, "You aren't my admin!")
			return
		}
		if !draftMail.isActive {
			_, err := b.Send(m.Sender, "No draft mail.\nPlease use /new to create one.")
			logError(err)
			return
		}
		body := m.Text
		if len(body) < 5 {
			_, err := b.Send(m.Sender, "Empty body!")
			logError(err)
			return
		}
		body = body[5:]
		draftMail.body = body
		log.Println(draftMail.body)
		_, err := b.Send(m.Sender, "Succeed.")
		logError(err)
	})

	b.Handle("/send", func(m *tb.Message) {
		if !m.Private() {
			return
		}
		if m.Sender.ID != config.UID {
			b.Send(m.Sender, "You aren't my admin!")
			return
		}
		if !draftMail.isActive {
			_, err := b.Send(m.Sender, "No draft mail.\nPlease use /new to create one.")
			logError(err)
			return
		}
		if len(draftMail.body) == 0 {
			_, err := b.Send(m.Sender, "Body is empty!")
			logError(err)
			return
		}
		if len(draftMail.subject) == 0 {
			_, err := b.Send(m.Sender, "Subject is empty!")
			logError(err)
			return
		}
		if len(draftMail.to) == 0 {
			_, err := b.Send(m.Sender, "Receiver's mail address is empty!")
			logError(err)
			return
		}
		err := smtpClient.Send([]string{draftMail.to}, draftMail.subject, draftMail.body)
		if err != nil {
			_, err = b.Send(m.Sender, "Catched an error while sending mail: "+err.Error())
		} else {
			_, err = b.Send(m.Sender, "Succeed.")
		}
		logError(err)
		draftMail.isActive = false
	})

	b.Handle(tb.OnText, func(m *tb.Message) {
		if !m.Private() {
			return
		}
		if m.Sender.ID != config.UID {
			b.Send(m.Sender, "You aren't my admin!")
			return
		}
		log.Println(m.Text)
	})

	_, err = c.UpdateMailCount()
	logError(err)

	go listen(c, b, config.UID)

	b.Start()
}

func listen(c *imap.Client, b *tb.Bot, uid int) {
	user := &tb.User{ID: uid, IsBot: false}
	d := time.Duration(time.Minute * 4)
	t := time.NewTicker(d)
	defer t.Stop()

	for {
		<-t.C
		l, r, err := c.GetNewMailRanges()

		if err != nil {
			log.Println(err)
		}

		if l == 0 {
			continue
		}
		for i := l; i <= r; i++ {
			text, _, err := c.ReadMail(i)
			if err != nil {
				log.Println(err)
			} else {
				_, err = b.Send(user, omitText(text))
				if err != nil {
					log.Println(err)
				}
			}
		}
	}
}

func omitText(text string) string {
	if len(text) > 4000 {
		return text[0:4000] + "\n<--Omit the following-->"
	} else {
		return text
	}
}

func logError(e error) {
	if e != nil {
		log.Println(e)
	}
}
