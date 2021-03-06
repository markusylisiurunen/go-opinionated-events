package opinionatedevents

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type sendableMeta struct {
	UUID      string    `json:"uuid" validate:"required"`
	Timestamp time.Time `json:"timestamp" validate:"required"`
}

type sendable struct {
	Name    string       `json:"name" validate:"required"`
	Payload []byte       `json:"payload"`
	Meta    sendableMeta `json:"meta" validate:"required"`
}

type payloadable interface {
	MarshalPayload() ([]byte, error)
	UnmarshalPayload([]byte) error
}

type messageMeta struct {
	uuid      string
	timestamp time.Time
}

type Message struct {
	name    string
	payload []byte
	meta    messageMeta
}

func (m *Message) Name() string {
	return m.name
}

func (m *Message) UUID() string {
	return m.meta.uuid
}

func (m *Message) Timestamp() time.Time {
	return m.meta.timestamp
}

func (msg *Message) Payload(payload payloadable) error {
	return payload.UnmarshalPayload(msg.payload)
}

func (msg *Message) MarshalJSON() ([]byte, error) {
	return json.Marshal(
		sendable{
			Name:    msg.name,
			Payload: msg.payload,
			Meta:    sendableMeta{UUID: msg.meta.uuid, Timestamp: msg.meta.timestamp.UTC()},
		},
	)
}

func NewMessage(name string, payload payloadable) (*Message, error) {
	pattern := "^[a-zA-Z0-9_\\-]+\\.[a-zA-Z0-9_\\-]+$"
	if matched, _ := regexp.MatchString(pattern, name); !matched {
		return nil, fmt.Errorf("name must match the pattern: %s", pattern)
	}

	message := &Message{
		name:    name,
		payload: nil,
		meta:    messageMeta{uuid: uuid.New().String(), timestamp: time.Now().UTC()},
	}

	if payload != nil {
		data, err := payload.MarshalPayload()
		if err != nil {
			return nil, err
		}
		message.payload = data
	}

	return message, nil
}

func newMessageFromSendable(data []byte) (*Message, error) {
	sendable := &sendable{}

	if err := json.Unmarshal(data, sendable); err != nil {
		return nil, err
	}

	if err := validator.New().Struct(sendable); err != nil {
		return nil, err
	}

	message := &Message{
		name:    sendable.Name,
		payload: sendable.Payload,
		meta:    messageMeta{uuid: sendable.Meta.UUID, timestamp: sendable.Meta.Timestamp},
	}

	return message, nil
}
