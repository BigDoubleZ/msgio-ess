package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	uuid "github.com/satori/go.uuid"
	"github.com/streadway/amqp"
)

type routeConfig struct {
	pattern     *regexp.Regexp
	handlerFunc func(http.ResponseWriter, *http.Request)
}

type responseData map[string]interface{}

var routes = []routeConfig{
	{handlerFunc: acceptMessage, pattern: regexp.MustCompile(`^/notifs/$`)},
	{handlerFunc: showMessageStatus, pattern: regexp.MustCompile(`^/notifs/(\d+)$`)},
	{handlerFunc: showMessageList, pattern: regexp.MustCompile(`^/notifs/\?page=(\d+)(&per_page=(\d+))?$`)},
	{handlerFunc: showAPISpecs, pattern: regexp.MustCompile(`^/notifs/docs$`)},
	// {handlerFunc: notFound, pattern: regexp.MustCompile(`.*`)},
}

var conn *amqp.Connection

func main() {

	var err error // FIXME make mq dealer
	conn, err = amqp.Dial(os.Getenv("MQ_URL"))
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	http.HandleFunc("/notifs/", mainHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	urlString := r.URL.String()
	log.Printf("Request: [%s]", urlString)

	found := false
	for _, route := range routes {
		if route.pattern.MatchString(urlString) {
			route.handlerFunc(w, r)
			found = true
			break
		}
	}

	if !found {
		log.Println("Not found.")
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

type essRequest map[string]interface{}

func acceptMessage(w http.ResponseWriter, r *http.Request) {
	log.Printf("acceptMessage")

	if r.Method != "POST" {
		http.Error(w, "Wrong method", http.StatusMethodNotAllowed)
		return
	}

	if r.ContentLength == 0 {
		http.Error(w, "Empty request", http.StatusBadRequest)
		return
	}

	requestData := essRequest{}
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Wrong request", http.StatusBadRequest)
		return
	}

	requestData["id"] = uuid.NewV4()

	fmt.Println(requestData)
	requestMsg, err := json.Marshal(requestData)
	if err != nil {
		http.Error(w, "Error parsing request", http.StatusBadRequest)
		return
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Printf("[!] mq: Failed to create channel: %s", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer ch.Close()

	ch.Confirm(false)
	confirms := ch.NotifyPublish(make(chan amqp.Confirmation))

	q, err := ch.QueueDeclare("ess", false, false, false, false, nil)
	if err != nil {
		log.Printf("[!] mq: Failed to declare a queue: %s", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// TODO request validation

	err = ch.Publish("", q.Name, false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(requestMsg),
		})
	if err != nil {
		log.Printf("[!] mq: Failed to declare a queue: %s", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	log.Printf("[ ] mq: Sent %s", requestData["id"])

	if confirmed := <-confirms; confirmed.Ack {
		w.WriteHeader(http.StatusAccepted)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(requestData)
	} else {
		log.Printf("[!] mq: Publish rejected: %s", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
}

func showMessageStatus(w http.ResponseWriter, r *http.Request) {
	log.Printf("showMessageStatus")
	// validate id
	// get message from db
	// send response
}

func showMessageList(w http.ResponseWriter, r *http.Request) {
	log.Printf("showMessageList")
	// validate params
	// get list from db
	// prepare headers
	// send response
	// send not found if none
}

func showAPISpecs(w http.ResponseWriter, r *http.Request) {
	log.Printf("showAPISpecs")
	// send static api specs?
}

func notFound(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not found", http.StatusNotFound)
	log.Println("Not found")
}

// func sendResponse(w http.ResponseWriter, data responseData) {
// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(data)
// }
