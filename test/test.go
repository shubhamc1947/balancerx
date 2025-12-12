package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	port := []string{"http://localhost:8080", "http://localhost:8080", "http://localhost:8080", "http://localhost:8080", "http://localhost:8080", "http://localhost:8080", "http://localhost:8080", "http://localhost:8080", "http://localhost:8080", "http://localhost:8080", "http://localhost:8080", "http://localhost:8080", "http://localhost:8080", "http://localhost:8080", "http://localhost:8080"}
	for i := 0; i < 10; i++ {
		for _, p := range port {
			go func(p string) {
				client := http.Client{
					Timeout: 2 * time.Second,
				}
				resp, err := client.Get(p)

				if err != nil {
					return
				}
				defer resp.Body.Close()
				fmt.Println("Success:", p, "Status:", resp)

			}(p)
		}
	}
	time.Sleep(5 * time.Second)

}
