package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// We will spin up 3 servers on different ports
	ports := []string{"8081", "8082", "8083"}

	for _, port := range ports {
		go func(p string) {
			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, "Hello from Backend Server on Port: %s\n", p)
			})

			log.Printf("Backend server started on port %s", p)
			log.Fatal(http.ListenAndServe(":"+p, mux))
		}(port)
	}

	// Block main thread forever
	select {}
}
