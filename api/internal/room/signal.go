package room

import "encoding/json"

type Incoming struct {
	Type    string          `json:"type"`
	To      string          `json:"to"`
	Payload json.RawMessage `json:"payload"`
}

type OutgoingSignal struct {
	Type    string          `json:"type"`
	From    string          `json:"from"`
	Payload json.RawMessage `json:"payload"`
}

func ResolveSignalTo(to, presenterID string) string {
	if to == "presenter" {
		return presenterID
	}
	return to
}

func RelaySignal(raw []byte, fromID string) ([]byte, string, bool) {
	var in Incoming
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, "", false
	}
	if in.Type != "signal" || in.To == "" {
		return nil, "", false
	}
	out, err := json.Marshal(OutgoingSignal{Type: "signal", From: fromID, Payload: in.Payload})
	if err != nil {
		return nil, "", false
	}
	return out, in.To, true
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{"type":"internal_error"}`)
	}
	return b
}
