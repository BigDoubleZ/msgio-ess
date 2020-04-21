package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_NotFound(t *testing.T) {

	req, err := http.NewRequest("GET", "/notifs/random_url", nil)
	if err != nil {
		t.Fatal(err)
	}

	checkErr := checkHTTPStatus(req, http.StatusNotFound)
	if checkErr != nil {
		t.Error(checkErr)
	}
}

func Test_AcceptMsg_WrongMethod(t *testing.T) {

	req, err := http.NewRequest("GET", "/notifs/", nil)
	if err != nil {
		t.Fatal(err)
	}

	checkErr := checkHTTPStatus(req, http.StatusMethodNotAllowed)
	if checkErr != nil {
		t.Error(checkErr)
	}
}

func Test_AcceptMsg_EmptyRequestBody(t *testing.T) {

	req, err := http.NewRequest("POST", "/notifs/", nil)
	if err != nil {
		t.Fatal(err)
	}

	checkErr := checkHTTPStatus(req, http.StatusBadRequest)
	if checkErr != nil {
		t.Error(checkErr)
	}
}

func Test_AcceptMsg_WrongRequestFormat(t *testing.T) {

	req, err := http.NewRequest("POST", "/notifs/",
		strings.NewReader("some text string"))
	if err != nil {
		t.Fatal(err)
	}

	checkErr := checkHTTPStatus(req, http.StatusBadRequest)
	if checkErr != nil {
		t.Error(checkErr)
	}
}

func Test_AcceptMsg_ValidRequest(t *testing.T) {

	params := map[string]interface{}{
		"sender":  "sender_A",
		"to":      []string{"bar@example.ru", "jar@example.ru"},
		"subject": "Тема уведомления",
		"message": "Текст уведомления",
	}

	body, _ := json.Marshal(params)

	req, err := http.NewRequest("POST", "/notifs/",
		bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(mainHandler)

	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusAccepted {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusAccepted)
	}
}

// func TestGetEntryByIDNotFound(t *testing.T) {
// 	req, err := http.NewRequest("GET", "/entry", nil)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	q := req.URL.Query()
// 	q.Add("id", "123")
// 	req.URL.RawQuery = q.Encode()
// 	rr := httptest.NewRecorder()
// 	handler := http.HandlerFunc(notFound)
// 	handler.ServeHTTP(rr, req)
// 	if status := rr.Code; status == http.StatusOK {
// 		t.Errorf("handler returned wrong status code: got %v want %v",
// 			status, http.StatusBadRequest)
// 	}
// }

///

func checkHTTPStatus(req *http.Request, statusExpected int) (err error) {

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(mainHandler)

	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != statusExpected {
		return fmt.Errorf(
			"Wrong http status: got %v, expected %v",
			status, statusExpected)
	}

	return nil

}
