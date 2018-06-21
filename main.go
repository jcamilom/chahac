package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/csv"
	"flag"
	"fmt"
	"io/ioutil"
	"net/smtp"
	"os"
	"strings"
	"syscall"
	"text/template"

	"golang.org/x/crypto/ssh/terminal"
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
	Nickname   string
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

// Ask the user for user and password through console and returns them
func getCredentials() (string, string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nUsername: ")
	username, _ := reader.ReadString('\n')

	fmt.Print("Password: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	eCheck("there was an error while getting the password", err)
	password := string(bytePassword)

	// TrimSpace removes all leading and trailing white spaces, including '\n'
	return strings.TrimSpace(username), password
}

func main() {

	/* ARGUMENTS PARSING */

	messageFilename := flag.String("msg", "message.txt", "Text template with the message in TXT format")
	contactsFilename := flag.String("for", "recipients.csv", "CSV file with the recipient's information")
	msgSubject := flag.String("sub", "Hello {{.Firstname}}", "Mail's subject template")
	serverHost := flag.String("host", "smtp.gmail.com", "Mail server host")
	serverPort := flag.String("port", "465", "Mail server port")
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
			Nickname:   recipient[4],
			Email:      recipient[5],
			Country:    recipient[6],
			C1:         recipient[7],
			C2:         recipient[8],
			C3:         recipient[9],
			C4:         recipient[10],
			C5:         recipient[11],
		}
	}

	/* PART TWO -> MESSAGE AND SUBJECT TEMPLATE */

	// Import the template's content
	msgContent, err := ioutil.ReadFile("files/" + *messageFilename)
	eCheck("there was an error with the file "+*messageFilename, err)

	// Create a new template and parse the message into it
	msgT := template.Must(template.New("message").Parse(string(msgContent)))
	// The same for the email subjet
	subjectT := template.Must(template.New("subject").Parse(*msgSubject))

	// Shows a message example when templates applied
	if len(recipients) > 0 {
		fmt.Print("************************************************************\n")
		fmt.Print("Preview of the generated message")
		fmt.Print("\n************************************************************\n")

		// Buffer to store the executed templates
		var b1, b2 bytes.Buffer

		// 	Add "subject" from template
		err = subjectT.Execute(&b1, recipients[0])
		eCheck("invalid 'subject' template", err)

		// Add the message from template
		err = msgT.Execute(&b2, recipients[0])
		eCheck("invalid 'message' template", err)

		message := ""
		message += fmt.Sprintf("From: ...\r\n")
		message += fmt.Sprintf("To: %s\r\n", recipients[0].Email)
		message += fmt.Sprintf("Subject: %s\r\n", b1.String())
		message += "\r\n" + b2.String()

		fmt.Print(message)
		fmt.Print("\n************************************************************\n")
		fmt.Print("Do you want to continue? (yes/no): ")

		reader := bufio.NewReader(os.Stdin)
		char, _, err := reader.ReadRune()
		eCheck("there was a problem reading your choice", err)

		fmt.Print("************************************************************\n")

		switch char {
		case 'y':
			break
		case 'n':
			fmt.Print("leaving...\n\n")
			os.Exit(1)
			break
		default:
			fmt.Print("Invalid option, leaving...\n\n")
			os.Exit(1)
			break
		}

	} else {
		fmt.Print("************************************************************\n\n")
		fmt.Print("Error: there are no entries in " + *contactsFilename)
		fmt.Print(".\n\n************************************************************\n")
		os.Exit(1)
	}

	/* PART THREE -> SETUP THE SMTP CONNECTION */

	// Get username and password from user input
	username, password := getCredentials()

	// Common mail fields
	mail := Mail{}
	mail.senderId = username

	smtpServer := SmtpServer{host: *serverHost, port: *serverPort}

	//build an auth
	auth := smtp.PlainAuth("", mail.senderId, password, smtpServer.host)

	// Gmail will reject connection if it's not secure
	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         smtpServer.host,
	}

	conn, err := tls.Dial("tcp", smtpServer.ServerName(), tlsconfig)
	eCheck("there was an error while dialing to the mail server", err)

	client, err := smtp.NewClient(conn, smtpServer.host)
	eCheck("there was an error whith the mail server", err)

	// Use Auth
	err = client.Auth(auth)
	eCheck("invalid username - password combination", err)

	/* PART FOUR -> CREATE AND SEND THE MESSAGE FOR EACH RECIPIENT */

	// Buffer to store the executed template
	var b bytes.Buffer
	l := len(recipients)

	fmt.Print("\nSending mails...")

	// Iterates over the recipients
	for i, v := range recipients {

		fmt.Printf("\n(%v/%v) %v... ", i+1, l, v.Email)

		// Add "to"
		mail.toId = v.Email

		// Add "subject" from template
		b.Reset() // Clears the buffer
		err = subjectT.Execute(&b, v)
		eCheck("invalid 'subject' template", err)
		mail.subject = b.String()

		// Add the message from template
		b.Reset() // Clears the buffer
		err = msgT.Execute(&b, v)
		eCheck("invalid 'message' template", err)
		mail.body = b.String()

		// Build the message
		messageBody := mail.BuildMessage()
		/* fmt.Print(messageBody)
		fmt.Print("\n=========================\n") */

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

		fmt.Print("[ok]")

	}

	client.Quit()

	fmt.Print("\n\nMails sent successfully.\n\n")

}
