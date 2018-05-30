package main

import (
	"crypto/tls"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"log"
	"net/smtp"
	"strings"
)

type Mail struct {
	senderId string
	toIds    []string
	subject  string
	body     string
}

type SmtpServer struct {
	host string
	port string
}

// ServerName returns the server name.
func (s *SmtpServer) ServerName() string {
	return s.host + ":" + s.port
}

// BuildMessage joins the header and the body of the message.
func (mail *Mail) BuildMessage() string {
	message := ""
	message += fmt.Sprintf("From: %s\r\n", mail.senderId)
	if len(mail.toIds) > 0 {
		message += fmt.Sprintf("To: %s\r\n", strings.Join(mail.toIds, ";"))
	}

	message += fmt.Sprintf("Subject: %s\r\n", mail.subject)
	message += "\r\n" + mail.body

	return message
}

// Most functions need to be checked for errors.
func eCheck(e error) {
	if e != nil {
		log.Panic(e)
	}
}

func main() {

	mail := Mail{}
	mail.senderId = "***@gmail.com"
	mail.subject = "This is the email subject"

	// First at all, import the recipients
	recipientsContent, err := ioutil.ReadFile("files/recipients.csv")
	eCheck(err)

	r := csv.NewReader(strings.NewReader(string(recipientsContent)))

	records, err := r.ReadAll()
	eCheck(err)

	records = records[1:] // Slices the first elements (file's header)
	mail.toIds = make([]string, len(records))
	for i, value := range records {
		mail.toIds[i] = value[2]
	}

	msgContent, err := ioutil.ReadFile("files/message.txt")
	eCheck(err)

	//fmt.Printf("File contents: %s", msgContent)

	mail.body = string(msgContent)

	//fmt.Printf(mail.body)

	messageBody := mail.BuildMessage()

	smtpServer := SmtpServer{host: "smtp.gmail.com", port: "465"}

	log.Println(smtpServer.host)
	//build an auth
	auth := smtp.PlainAuth("", mail.senderId, "password", smtpServer.host)

	// Gmail will reject connection if it's not secure
	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         smtpServer.host,
	}

	conn, err := tls.Dial("tcp", smtpServer.ServerName(), tlsconfig)
	if err != nil {
		log.Panic(err)
	}

	client, err := smtp.NewClient(conn, smtpServer.host)
	if err != nil {
		log.Panic(err)
	}

	// step 1: Use Auth
	if err = client.Auth(auth); err != nil {
		log.Panic(err)
	}

	// step 2: add all from and to
	if err = client.Mail(mail.senderId); err != nil {
		log.Panic(err)
	}
	for _, k := range mail.toIds {
		if err = client.Rcpt(k); err != nil {
			log.Panic(err)
		}
	}

	// Data
	w, err := client.Data()
	if err != nil {
		log.Panic(err)
	}

	_, err = w.Write([]byte(messageBody))
	if err != nil {
		log.Panic(err)
	}

	err = w.Close()
	if err != nil {
		log.Panic(err)
	}

	client.Quit()

	log.Println("Mails sent successfully")

}
