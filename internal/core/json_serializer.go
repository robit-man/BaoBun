package core

import (
	"encoding/json"

	"github.com/baoswarm/baobun/pkg/protocol"
)

type JSONSerializer struct{}

func NewJSONSerializer() *JSONSerializer {
	return &JSONSerializer{}
}

func (j *JSONSerializer) MarshalPeerMessage(msg *protocol.PeerMessage) ([]byte, error) {
	return json.Marshal(msg)
}

func (j *JSONSerializer) UnmarshalPeerMessage(data []byte, msg *protocol.PeerMessage) error {
	return json.Unmarshal(data, msg)
}

func (j *JSONSerializer) MarshalHandshakePayload(p *protocol.HandshakePayload) ([]byte, error) {
	return json.Marshal(p)
}

func (j *JSONSerializer) UnmarshalHandshakePayload(data []byte, p *protocol.HandshakePayload) error {
	return json.Unmarshal(data, p)
}

func (j *JSONSerializer) MarshalBitfieldPayload(p *protocol.BitfieldPayload) ([]byte, error) {
	return json.Marshal(p)
}

func (j *JSONSerializer) UnmarshalBitfieldPayload(data []byte, p *protocol.BitfieldPayload) error {
	return json.Unmarshal(data, p)
}

func (j *JSONSerializer) MarshalHavePayload(p *protocol.HavePayload) ([]byte, error) {
	return json.Marshal(p)
}

func (j *JSONSerializer) UnmarshalHavePayload(data []byte, p *protocol.HavePayload) error {
	return json.Unmarshal(data, p)
}

func (j *JSONSerializer) MarshalTransferRequestPayload(p *protocol.TransferRequestPayload) ([]byte, error) {
	return json.Marshal(p)
}

func (j *JSONSerializer) UnmarshalTransferRequestPayload(data []byte, p *protocol.TransferRequestPayload) error {
	return json.Unmarshal(data, p)
}

func (j *JSONSerializer) MarshalTransferPayload(p *protocol.TransferPayload) ([]byte, error) {
	return json.Marshal(p)
}

func (j *JSONSerializer) UnmarshalTransferPayload(data []byte, p *protocol.TransferPayload) error {
	return json.Unmarshal(data, p)
}
