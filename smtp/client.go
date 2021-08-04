package smtp

import (
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

type Client struct {
	serverAddress string
	serverHost string
	email         string
	password      string
}

func NewClient(serverAddress string, serverHost string,email string, password string) *Client {
	return &Client{serverAddress: serverAddress, serverHost: serverHost, email: email, password: password}
}

func (C *Client) Send(to []string, subject string, body string) error {
	body=strings.Replace(body,"\r\n","\n",-1)
	body=strings.Replace(body,"\n","\r\n",-1)
	auth:=NewLoginAuth(C.email,C.password)
	err:=SendMail(C.serverAddress,auth,C.email,to,subject, []byte(body))
	return err
}

func SendMail(addr string, a smtp.Auth, from string, to []string, subject string,msg []byte) error {
	c, err := smtp.Dial(addr)
	host, _, _ := net.SplitHostPort(addr)
	if err != nil {
		return errors.New("call dial: "+err.Error())
	}
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); ok {
		config := &tls.Config{ServerName: host, InsecureSkipVerify: true}
		if err = c.StartTLS(config); err != nil {
			return errors.New("call start tls: "+err.Error())
		}
	}

	if a != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err = c.Auth(a); err != nil {
				return errors.New("check auth with err: " + err.Error())
			}
		}
	}

	if err = c.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}

	header := make(map[string]string)
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/plain; charset=\"utf-8\""
	header["Content-Transfer-Encoding"] = "base64"
	header["From"] = from
	toText:=""
	for i, v:=range to {
		toText+=v
		if i!=len(to) {
			toText+=", "
		}
	}
	header["To"] = toText
	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + base64.StdEncoding.EncodeToString(msg)
	_, err = w.Write([]byte(message))

	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	return c.Quit()
}
