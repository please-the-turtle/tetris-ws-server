package main

import (
	"log/slog"
	"net/http"
	"os"
)

func main() {
	logger := NewLogger(slog.LevelDebug)
	slog.SetDefault(logger)

	wsServer := NewWebsocketServer()
	go wsServer.Run()
	http.HandleFunc("/ws", wsServer.ServeWs)

	err := http.ListenAndServe(":1213", nil)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
