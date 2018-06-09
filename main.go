package main

import (
	"bytes"
	"crypto/tls"
	"encoding/csv"
	"flag"
	"fmt"
	"io/ioutil"
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

// Check for errors and print a custom message
func eCheck(msg string, e error) {
	if e != nil {
		fmt.Print("************************************************************\n\n")
		fmt.Print("Error: " + msg)
		fmt.Print(".\n\n************************************************************\n")
		panic(e)
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
	eCheck("there was an error opening the file "+*contactsFilename, err)

	r := csv.NewReader(strings.NewReader(string(recipientsContent)))

	records, err := r.ReadAll()
	eCheck("there was an error reading the file "+*contactsFilename, err)

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
	eCheck("there was an error with the file "+*messageFilename, err)

	// Create a new template and parse the message into it
	msgT := template.Must(template.New("message").Parse(string(msgContent)))
	// The same for the email subjet
	subjectT := template.Must(template.New("subject").Parse(*msgSubject))

	/* PART THREE -> SETUP THE SMTP CONNECTION */

	// Common mail fields
	mail := Mail{}
	mail.senderId = "***@gmail.com"

	smtpServer := SmtpServer{host: "smtp.gmail.com", port: "465"}

	fmt.Println(smtpServer.host)
	//build an auth
	auth := smtp.PlainAuth("", mail.senderId, "password", smtpServer.host)

	// Gmail will reject connection if it's not secure
	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         smtpServer.host,
	}

	conn, err := tls.Dial("tcp", smtpServer.ServerName(), tlsconfig)
	eCheck("there was an error while dialing to the mail server", err)

	client, err := smtp.NewClient(conn, smtpServer.host)
	eCheck("there was an error whit the mail server", err)

	// Use Auth
	err = client.Auth(auth)
	eCheck("invalid username - password combination", err)

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
		err = subjectT.Execute(&b2, v)
		eCheck("invalid 'subject' template", err)
		mail.subject = b2.String()

		// Add the message from template
		err = msgT.Execute(&b, v)
		eCheck("invalid 'message' template", err)
		mail.body = b.String()

		// Build the message
		messageBody := mail.BuildMessage()
		fmt.Print(messageBody)
		fmt.Print("\n=========================\n")

		// Add "from" header
		err = client.Mail("***@gmail.com")
		eCheck("there was an error adding the 'from' header", err)

		// Add "to" header
		err = client.Rcpt(v.Email)
		eCheck("there was an error adding the 'to' header", err)

		// Data
		w, err := client.Data()
		eCheck("there was an error adding the message content", err)

		_, err = w.Write([]byte(messageBody))
		eCheck("there was an error writting the message", err)

		err = w.Close()
		eCheck("there was an error closing the message's writter", err)

		// Clear the buffer
		b.Reset()
		b2.Reset()

	}

	client.Quit()

	fmt.Println("Mails sent successfully")

}
