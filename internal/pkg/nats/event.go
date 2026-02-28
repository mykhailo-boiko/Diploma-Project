package nats

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Source    string          `json:"source"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

func NewEvent(eventType, source string, data any) (Event, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return Event{}, fmt.Errorf("failed to marshal event data: %w", err)
	}

	return Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now().UTC(),
		Data:      raw,
	}, nil
}

func (e Event) DecodeData(target any) error {
	if err := json.Unmarshal(e.Data, target); err != nil {
		return fmt.Errorf("failed to decode event data: %w", err)
	}
	return nil
}

func (e Event) Encode() ([]byte, error) {
	raw, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("failed to encode event: %w", err)
	}
	return raw, nil
}

func DecodeEvent(data []byte) (Event, error) {
	var ev Event
	if err := json.Unmarshal(data, &ev); err != nil {
		return Event{}, fmt.Errorf("failed to decode event: %w", err)
	}
	return ev, nil
}
