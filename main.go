package main

import (
	"bytes"
	"crypto/tls"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"log"
	"net/smtp"
	"os"
	"strings"
	"text/template"
)

type Mail struct {
	senderId string
	toId     string
	subject  string
	body     string
}

type SmtpServer struct {
	host string
	port string
}

type Recipient struct {
	Firstname string
	Lastname  string
	Email     string
	Country   string
}

// ServerName returns the server name.
func (s *SmtpServer) ServerName() string {
	return s.host + ":" + s.port
}

// BuildMessage joins the header and the body of the message.
func (mail *Mail) BuildMessage() string {
	message := ""
	message += fmt.Sprintf("From: %s\r\n", mail.senderId)
	message += fmt.Sprintf("To: %s\r\n", mail.toId)
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

	/* ARGUMENTS CHECKING */

	args := os.Args[1:]
	argsLength := len(args)
	if argsLength != 2 {
		panic("Wrong usage. Chahac needs to be executed along two arguments. \n\nExample: ./chahac message.txt recipients.csv")
	}

	/* PART ONE -> RECIPIENTS */

	// Import the recipiens file and create the recipients
	var recipients []Recipient
	recipientsContent, err := ioutil.ReadFile("files/" + args[1])
	eCheck(err)

	r := csv.NewReader(strings.NewReader(string(recipientsContent)))

	records, err := r.ReadAll()
	eCheck(err)

	records = records[1:] // Slices the first element (file's header)
	recipients = make([]Recipient, len(records))
	for i, recipient := range records {
		recipients[i].Firstname = recipient[0]
		recipients[i].Lastname = recipient[1]
		recipients[i].Email = recipient[2]
		recipients[i].Country = recipient[3]
	}

	/* PART TWO -> MESSAGE TEMPLATE */

	// Import the template's content
	msgContent, err := ioutil.ReadFile("files/" + args[0])
	eCheck(err)

	// Create a new template and parse the message into it.
	t := template.Must(template.New("message").Parse(string(msgContent)))

	/* PART THREE -> SETUP THE SMTP CONNECTION */

	// Common mail fields
	mail := Mail{}
	mail.senderId = "***@gmail.com"

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

	// Use Auth
	if err = client.Auth(auth); err != nil {
		log.Panic(err)
	}

	/* PART FOUR -> CREATE AND SEND THE MESSAGE FOR EACH RECIPIENT */

	// Buffer to store the executed template
	var b bytes.Buffer

	// Iterates over the recipients
	for _, v := range recipients {

		/* fmt.Printf("%+v\n", v) */
		// Add "to"
		mail.toId = v.Email

		// 	Add "subject"
		mail.subject = "Hola " + v.Firstname

		// Execute the template
		err := t.Execute(&b, v)
		if err != nil {
			log.Println("executing template:", err)
		}

		// Add "body"
		mail.body = b.String()

		// Build the message
		messageBody := mail.BuildMessage()
		fmt.Print(messageBody)
		fmt.Print("\n=========================\n")

		// Add "from" header
		if err = client.Mail("***@gmail.com"); err != nil {
			log.Panic(err)
		}

		// Add "to" header
		if err = client.Rcpt(v.Email); err != nil {
			log.Panic(err)
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

		// Clear the buffer
		b.Reset()

	}

	client.Quit()

	log.Println("Mails sent successfully")

}
