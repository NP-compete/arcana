package main

import (
	"fmt"
	"net/http"

	"github.com/NP-compete/arcana/pkg/server"
)

func main() {
	httpSrv := server.New(server.Config{
		ServiceName: "agui",
		Port:        "8084",
	})

	httpSrv.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		fmt.Fprint(w, "data: {\"type\":\"RunStarted\"}\n\n")
	})

	httpSrv.ListenAndServe()
}
