/*
Favourites is a simple web server that manages a per-user list of favourite assets.
*/
package main

import (
	"log"
	"net/http"

	"github.com/parath/platform-go-challenge/internal/favourites"
	"github.com/parath/platform-go-challenge/internal/httpapi"
)

func main() {
	r := httpapi.NewServer(favourites.NewInMemoryStore())
	log.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
