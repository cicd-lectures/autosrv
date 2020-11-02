package main

import (
	"io/ioutil"
	"log"
	"net/http"
)

func handleNotification(w http.ResponseWriter, r *http.Request) {
	log.Println("Receiving notification")
	defer r.Body.Close()

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "unable to ready body", http.StatusInternalServerError)
		return
	}

	log.Println(string(b))

	w.WriteHeader(http.StatusOK)
}

func main() {
	http.Handle("/notification", http.HandlerFunc(handleNotification))

	log.Println("Listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
