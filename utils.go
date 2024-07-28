package main

import (
	"encoding/json"
	"log/slog"
)

func EncodeToJSON(v any) []byte {
	bytes, err := json.Marshal(v)
	if err != nil {
		slog.Error(err.Error(), slog.Any("target", v))
	}

	return bytes
}
