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
	"os"
	"strings"

	"github.com/streadway/amqp"
)

// ESSMessage - message recieved from MQ
type ESSMessage struct {
	ID      string   `json:"id"`
	Sender  string   `json:"sender"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Message string   `json:"message"`
}

var mqConn *amqp.Connection

func main() {

	var err error
	mqConn, err = amqp.Dial(os.Getenv("MQ_URL"))
	if err != nil {
		log.Fatalf("[f] Failed to connect to RabbitMQ: %s", err)
	}
	defer mqConn.Close()

	ch, err := mqConn.Channel()
	if err != nil {
		log.Fatalf("[f] Failed to open a channel: %s", err)
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
		log.Fatalf("[f] Failed to declare a queue: %s", err)
	}

	err = ch.Qos(1, 0, false)
	if err != nil {
		log.Fatalf("[f] Failed to set QOS: %s", err)
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("[ ] Received a message: %s", d.Body)
			dispatchMessage(d)
		}
	}()

	log.Printf("[*] Waiting for messages. To exit press CTRL+C")

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
		log.Println("[!] Error decoding message")
		msg.Ack(false)
		return
	}

	rec := model.ESSRec{
		ID:      msgData.ID,
		Sender:  msgData.Sender,
		Subject: msgData.Subject,
		Message: msgData.Subject,
		To:      strings.Join(msgData.To, ":"),
	}

	if err := model.AddRecord(rec); err != nil {
		log.Printf("[!] db: Failed to store record: %s", err)
		msg.Ack(false)
		return
	}

	msg.Ack(true)

	fmt.Println(msgData)

	from := mail.Address{Name: msgData.Sender, Address: "sender-no-reply@example.com"}

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

	// smtpServer := "smtp.msgio.com"
	// auth := smtp.PlainAuth(
	// 	"",
	// 	"test@user",
	// 	"testpassword",
	// 	smtpServer,
	// )

	// err = smtp.SendMail(
	// 	smtpServer+":25",
	// 	auth,
	// 	from.Address,
	// 	[]string{to.Address},
	// 	[]byte(message),
	// )
	// if err != nil {
	// 	log.Fatal(err)
	// }

	if err = model.SetRecordSent(msgData.ID, true); err != nil {
		log.Printf("[!] db: Error setting status: %s", err)
	}
}
