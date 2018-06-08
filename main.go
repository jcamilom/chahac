package main

import (
	"bytes"
	"crypto/tls"
	"encoding/csv"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/smtp"
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
	Firstname1 string
	Firstname2 string
	Firstname  string
	Lastname1  string
	Lastname2  string
	Lastname   string
	Email      string
	Country    string
	C1         string
	C2         string
	C3         string
	C4         string
	C5         string
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

	/* ARGUMENTS PARSING */

	messageFilename := flag.String("msg", "message.txt", "Text template with the message in TXT format")
	contactsFilename := flag.String("for", "recipients.csv", "CSV file with the recipient's information")
	msgSubject := flag.String("sub", "Hello {{.Firstname}}", "Mail's subject template")
	flag.Parse()

	/* PART ONE -> RECIPIENTS */

	// Import the recipiens file and create the recipients
	var recipients []Recipient
	recipientsContent, err := ioutil.ReadFile("files/" + *contactsFilename)
	eCheck(err)

	r := csv.NewReader(strings.NewReader(string(recipientsContent)))

	records, err := r.ReadAll()
	eCheck(err)

	records = records[1:] // Slices the first element (file's header)
	recipients = make([]Recipient, len(records))
	for i, recipient := range records {

		// Join names
		firstname := recipient[0]
		if name2 := recipient[1]; name2 != "" {
			firstname += " " + name2
		}

		// Join lastnames
		lastname := recipient[2]
		if lastname2 := recipient[3]; lastname2 != "" {
			lastname += " " + lastname2
		}

		// Create the recipient
		recipients[i] = Recipient{
			Firstname1: recipient[0],
			Firstname2: recipient[1],
			Firstname:  firstname,
			Lastname1:  recipient[2],
			Lastname2:  recipient[3],
			Lastname:   lastname,
			Email:      recipient[4],
			Country:    recipient[5],
			C1:         recipient[6],
			C2:         recipient[7],
			C3:         recipient[8],
			C4:         recipient[9],
			C5:         recipient[10],
		}
	}

	/* PART TWO -> MESSAGE AND TITLE TEMPLATE */

	// Import the template's content
	msgContent, err := ioutil.ReadFile("files/" + *messageFilename)
	eCheck(err)

	// Create a new template and parse the message into it.
	msgT := template.Must(template.New("message").Parse(string(msgContent)))
	// The same for the email subjet
	subjectT := template.Must(template.New("subject").Parse(*msgSubject))

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
	var b2 bytes.Buffer

	// Iterates over the recipients
	for _, v := range recipients {

		/* fmt.Printf("%+v\n", v) */
		// Add "to"
		mail.toId = v.Email

		// 	Add "subject" from template
		err := subjectT.Execute(&b2, v)
		if err != nil {
			log.Println("executing template:", err)
		}
		mail.subject = b2.String()

		// Add the message from template
		err = msgT.Execute(&b, v)
		if err != nil {
			log.Println("executing template:", err)
		}
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
		b2.Reset()

	}

	client.Quit()

	log.Println("Mails sent successfully")

}
