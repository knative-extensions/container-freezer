package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	go func() {
		// prints all the time in the background, even when no requests are being
		// processed - naughty!.
		for range time.Tick(500 * time.Millisecond) {
			fmt.Println("Ticking at:", time.Now().Format(time.ANSIC))
		}
	}()

	// simple handler, just sleeps 5 seconds - backgrond tasks allowed to run
	// while this is going.
	http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.(http.Flusher).Flush()
		time.Sleep(5 * time.Second)
		fmt.Fprintln(w, "Hello, this request will take five seconds..")
	}))
}
