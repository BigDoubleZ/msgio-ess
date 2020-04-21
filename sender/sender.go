package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/mail"
	"strings"

	"github.com/streadway/amqp"
)

var conn *amqp.Connection

func main() {

	var err error
	conn, err = amqp.Dial("amqp://test:test@localhost:5672")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"ess", // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	failOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
			dispatchMessage(d)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever

}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

type essRequest map[string]interface{}

func dispatchMessage(msg amqp.Delivery) {

	msgData := essRequest{}

	buf := bytes.NewBuffer(msg.Body)
	err := json.NewDecoder(buf).Decode(&msgData)
	if err != nil {
		log.Println("[!] Error decoding acceptor message")
		return
	}

	fmt.Println(msgData)

	// smtpServer := "smtp.msgio.com"
	// auth := smtp.PlainAuth(
	// 	"",
	// 	"test@user",
	// 	"testpassword",
	// 	smtpServer,
	// )

	from := mail.Address{Name: "sender", Address: "noreply@sender.com"}
	// to := mail.Address{Name: "test address", Address: "noreply@sender.com"}

	header := make(map[string]string)
	header["From"] = from.String()
	// header["To"] = to.String()
	header["Subject"] = encodeRFC2047("subject string")
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/plain; charset=\"utf-8\""
	header["Content-Transfer-Encoding"] = "base64"

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + base64.StdEncoding.EncodeToString([]byte("message text"))

	log.Println(message)

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

}

func encodeRFC2047(str string) string {
	addr := mail.Address{Address: str}
	return strings.Trim(addr.String(), " <>")
}
