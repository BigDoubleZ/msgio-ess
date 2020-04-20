package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/notifs", msgAccept)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func msgAccept(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{status: %s}`, "ok")

}
