package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"msgio-ess/model"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	uuid "github.com/satori/go.uuid"
	"github.com/streadway/amqp"
)

var reEmail = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

type ESSRequest struct {
	Sender  string   `json:"sender,ommitempty"`
	To      []string `json:"to"`
	Subject string   `json:"subject,ommitempty"`
	Message string   `json:"message"`
}

type ESSMessage struct {
	ID string
	ESSRequest
}

type ESSResponse struct {
	ID      string `json:"id"`
	Created string `json:"created_at"`
	Sent    string `json:"sent_status"`
	ESSRequest
}

type routeConfig struct {
	pattern     *regexp.Regexp
	handlerFunc func(http.ResponseWriter, *http.Request)
}

var showMessageRe = regexp.MustCompile(`(?i)^/notifs/([0-9a-f]{8}\-([0-9a-f]{4}\-){3}[0-9a-f]{12})$`)
var routes = []routeConfig{
	{handlerFunc: acceptMessage, pattern: regexp.MustCompile(`^/notifs/$`)},
	{handlerFunc: showMessageStatus, pattern: showMessageRe},
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

func acceptMessage(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		log.Printf("[!] api: acceptMessage: Wrong method")
		http.Error(w, "Wrong method", http.StatusMethodNotAllowed)
		return
	}

	if r.ContentLength == 0 {
		log.Printf("[!] api: acceptMessage: Empty request body")
		http.Error(w, "Empty request", http.StatusBadRequest)
		return
	}

	requestMsg, err := prepareMessage(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	msg, err := json.Marshal(requestMsg)
	if err != nil {
		log.Printf("[!] mq: Error encodign message: %s", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	err = ch.Publish("", q.Name, false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(msg),
		})
	if err != nil {
		log.Printf("[!] mq: Failed to declare a queue: %s", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	if confirmed := <-confirms; confirmed.Ack {
		log.Printf("[ ] mq: Sent %s", requestMsg.ID)
		w.WriteHeader(http.StatusAccepted)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"id": "%s"}`, requestMsg.ID)
	} else {
		log.Printf("[!] mq: Failed to publish message: %s", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
}

func showMessageStatus(w http.ResponseWriter, r *http.Request) {
	args := showMessageRe.FindAllStringSubmatch(r.URL.String(), -1)

	id := args[0][1]

	if len(id) < 32 {
		log.Printf("[!] api: showMessage: impossible regex mismatch")
		http.Error(w, "Server error", http.StatusInternalServerError)
	}

	msg, err := model.GetRecord(id)
	if err != nil {
		log.Printf("[!] api: showMessage: not found %s", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	log.Printf("%v", msg)

	response, err := recordsToResponse([]*model.ESSRec{msg})
	if err != nil {
		log.Printf("[!] api: showMessage: Error encoding message: %s", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", response)
}

func showMessageList(w http.ResponseWriter, r *http.Request) {
	log.Printf("showMessageList")

	page, err := extractParam(r, "page", 1)
	if err != nil {
		http.Error(w, "Wrong params", http.StatusBadRequest)
		return
	}

	pageSize, err := extractParam(r, "per_page", 20)
	if err != nil {
		http.Error(w, "Wrong params", http.StatusBadRequest)
		return
	}

	if pageSize > 1000 {
		log.Printf("[!] api: showMessageList: page size exceeds limit: %d", pageSize)
		http.Error(w, "Wrong params", http.StatusBadRequest)
		return
	}

	if pageSize < 1 {
		log.Printf("[!] api: showMessageList: page size exceeds limit: %d", pageSize)
		http.Error(w, "Wrong params", http.StatusBadRequest)
		return
	}

	msgCount, err := model.RecordCount()
	if err != nil {
		log.Printf("[!] api: showMessageList: Error getting message count")
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	log.Print(string(msgCount))

	msgList, err := model.GetRecordList(pageSize, pageSize*(page-1))
	if err != nil {
		log.Printf("[!] api: showMessageList: Error getting message list")
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	if len(msgList) < 1 {
		log.Printf("[ ] api: showMessageList: Empty message list")
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	response, err := recordsToResponse(msgList)
	if err != nil {
		log.Printf("[!] api: showMessage: Error encoding message: %s", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	maxPage := (msgCount / pageSize) + 1
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Total", strconv.Itoa(msgCount))
	w.Header().Set("X-Total-Pages", strconv.Itoa(maxPage))
	w.Header().Set("X-Per-Page", strconv.Itoa(pageSize))
	w.Header().Set("X-Page", strconv.Itoa(page))

	if page+1 <= maxPage {
		w.Header().Set("X-Next-Page", strconv.Itoa(page+1))
	}
	if page > 1 {
		w.Header().Set("X-Prev-Page", strconv.Itoa(page-1))
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "[%s]", response)
}

func recordsToResponse(recList []*model.ESSRec) (string, error) {

	parts := make([]string, 0, len(recList))

	for _, rec := range recList {
		essr := ESSResponse{}
		essr.ID = rec.ID
		essr.Sender = rec.Sender
		essr.Created = rec.Created
		essr.Subject = rec.Subject
		essr.Message = rec.Message
		essr.Sent = rec.Sent

		essr.To = strings.Split(rec.To, ":")

		chunk, err := json.Marshal(essr)
		if err != nil {
			return "", err
		}

		parts = append(parts, string(chunk))
		// response.Write(chunk)

	}

	return strings.Join(parts, ","), nil
}

func showAPISpecs(w http.ResponseWriter, r *http.Request) {
	log.Printf("showAPISpecs")
	// send static api specs?
}

func notFound(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not found", http.StatusNotFound)
	log.Println("Not found")
}

func prepareMessage(req *http.Request) (*ESSMessage, error) {

	msg := new(ESSMessage)

	err := json.NewDecoder(req.Body).Decode(msg)
	if err != nil {
		log.Printf("[!] req: Error parsing request: %s", err)
		return nil, errors.New("Valid request expected")
	}

	if len(msg.To) < 1 {
		log.Print("[!] req: Empty email list")
		return nil, errors.New("Non-empty address list expected")
	}

	for _, email := range msg.To {
		if len(email) > 254 || !reEmail.MatchString(email) {
			log.Print("[!] req: Invalid email address")
			return nil, errors.New("Invalid email address")
		}
	}

	msg.ID = uuid.NewV4().String()

	// fmt.Println(msg) // DBG

	return msg, err
}

func extractParam(r *http.Request, name string, init int) (int, error) {
	value := init

	field, ok := r.URL.Query()[name]
	if !ok || len(field[0]) < 1 {
		log.Printf("[ ] api: extractParam: Empty [%s] parameter", name)
		return value, nil
	}
	value, err := strconv.Atoi(field[0])
	if err != nil {
		log.Printf("[!] api: extractParam: Invalid [%s] parameter: [%s]", name, field[0])
		return value, err
	}

	return value, nil
}
