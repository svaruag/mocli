package outfmt

import (
	"encoding/json"
	"fmt"
	"io"
)

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

type ErrorEnvelope struct {
	Error ErrorPayload `json:"error"`
}

func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func WriteError(w io.Writer, jsonMode bool, payload ErrorPayload) error {
	if jsonMode {
		return WriteJSON(w, ErrorEnvelope{Error: payload})
	}

	if payload.Hint != "" {
		_, err := fmt.Fprintf(w, "error: %s\nhint: %s\n", payload.Message, payload.Hint)
		return err
	}
	_, err := fmt.Fprintf(w, "error: %s\n", payload.Message)
	return err
}
