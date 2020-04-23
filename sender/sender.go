package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"msgio-ess/model"
	"net/mail"
	"net/smtp"
	"os"
	"strings"

	"github.com/streadway/amqp"
)

type ESSMessage struct {
	ID      string   `json:"id"`
	Sender  string   `json:"sender"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Message string   `json:"message"`
}

var (
	cfgSenderAddress  = os.Getenv("MAIL_SENDERADDRESS")
	cfgServerURL      = os.Getenv("MAIL_SERVERURL")
	cfgServerPort     = os.Getenv("MAIL_SERVERPORT")
	cfgServerUser     = os.Getenv("MAIL_USER")
	cfgServerPassword = os.Getenv("MAIL_PASSWORD")
)

var mqConn *amqp.Connection

func main() {

	model.Setup()

	var err error
	mqConn, err = amqp.Dial(os.Getenv("MQ_URL"))
	if err != nil {
		log.Fatalf("[f] mq: Failed to connect to RabbitMQ: %s", err)
	}
	defer mqConn.Close()

	ch, err := mqConn.Channel()
	if err != nil {
		log.Fatalf("[f] mq: Failed to open a channel: %s", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		os.Getenv("MQ_QUEUE"), // name
		false,                 // durable
		false,                 // delete when unused
		false,                 // exclusive
		false,                 // no-wait
		nil,                   // arguments
	)
	if err != nil {
		log.Fatalf("[f] mq: Failed to declare a queue: %s", err)
	}

	err = ch.Qos(1, 0, false)
	if err != nil {
		log.Fatalf("[f] mq: Failed to set QOS: %s", err)
	}

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("[f] mq: Failed to register a consumer: %s", err)
	}

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("[ ] processing message: %s", d.Body)
			dispatchMessage(d)
		}
	}()

	log.Printf("[*] Waiting for messages")

	<-forever
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func dispatchMessage(msg amqp.Delivery) {

	msgData := ESSMessage{}

	buf := bytes.NewBuffer(msg.Body)
	err := json.NewDecoder(buf).Decode(&msgData)
	if err != nil {
		log.Println("[!] mq: Error decoding message", err)
		msg.Ack(false)
		return
	}

	rec := model.ESSRec{
		ID:      msgData.ID,
		Sender:  msgData.Sender,
		Subject: msgData.Subject,
		Message: msgData.Message,
		To:      strings.Join(msgData.To, ":"),
	}

	if err := model.AddRecord(rec); err != nil {
		log.Printf("[!] db: Failed to store record: %s", err)
		msg.Ack(false)
		return
	}

	msg.Ack(true)

	fmt.Println(msgData)

	from := mail.Address{Name: msgData.Sender, Address: cfgSenderAddress}

	header := make(map[string]string)
	header["From"] = from.String()
	header["Subject"] = mime.BEncoding.Encode("utf-8", msgData.Subject)
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/plain; charset=\"utf-8\""
	header["Content-Transfer-Encoding"] = "base64"

	message := ""
	for field, val := range header {
		message += fmt.Sprintf("%s: %s\r\n", field, val)
	}
	message += "\r\n" + base64.StdEncoding.EncodeToString([]byte(msgData.Message))

	log.Println(message) // DBG

	auth := smtp.PlainAuth(
		"",
		cfgServerUser,
		cfgServerPassword,
		cfgServerURL,
	)

	err = smtp.SendMail(
		cfgServerURL+":"+cfgServerPort,
		auth,
		from.Address,
		msgData.To,
		[]byte(message),
	)
	if err != nil {
		log.Printf("[!] mail: Sending failed: %s", err)
		return
	}

	if err = model.SetRecordSent(msgData.ID, true); err != nil {
		log.Printf("[!] db: Error setting status: %s", err)
	}
	log.Printf("[*] mail: Sending completed: %s", msgData.ID)
}
