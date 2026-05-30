package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8089"
	}

	store := NewPredictionStore()
	predictor := NewPredictor(store)
	srv := NewServer(store, predictor)

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	http.HandleFunc("/api/v1/predict", srv.corsMiddleware(srv.handlePredict))
	http.HandleFunc("/api/v1/predictions/", srv.corsMiddleware(srv.handleGetPrediction))
	http.HandleFunc("/api/v1/calibrate", srv.corsMiddleware(srv.handleCalibrate))
	http.HandleFunc("/api/v1/stats", srv.corsMiddleware(srv.handleStats))

	log.Printf("arcana-oracle starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
