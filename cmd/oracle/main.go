package main

import (
	"github.com/NP-compete/arcana/pkg/server"
)

func main() {
	httpSrv := server.New(server.Config{
		ServiceName: "oracle",
		Port:        "8089",
	})

	store := NewPredictionStore()
	predictor := NewPredictor(store)
	srv := NewServer(store, predictor)

	httpSrv.HandleFunc("/api/v1/predict", srv.corsMiddleware(srv.handlePredict))
	httpSrv.HandleFunc("/api/v1/predictions/", srv.corsMiddleware(srv.handleGetPrediction))
	httpSrv.HandleFunc("/api/v1/calibrate", srv.corsMiddleware(srv.handleCalibrate))
	httpSrv.HandleFunc("/api/v1/stats", srv.corsMiddleware(srv.handleStats))

	httpSrv.ListenAndServe()
}
