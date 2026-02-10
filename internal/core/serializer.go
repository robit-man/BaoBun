package core

import "github.com/baoswarm/baobun/pkg/protocol"

type Serializer interface {
	// MarshalPeerMessage serializes a PeerMessage
	MarshalPeerMessage(msg *protocol.PeerMessage) ([]byte, error)

	// UnmarshalPeerMessage deserializes bytes into a PeerMessage
	UnmarshalPeerMessage(data []byte, msg *protocol.PeerMessage) error

	// MarshalHandshakePayload serializes a HandshakePayload
	MarshalHandshakePayload(p *protocol.HandshakePayload) ([]byte, error)

	// UnmarshalHandshakePayload deserializes bytes into a HandshakePayload
	UnmarshalHandshakePayload(data []byte, p *protocol.HandshakePayload) error

	// MarshalBitfieldPayload serializes a BitfieldPayload
	MarshalBitfieldPayload(p *protocol.BitfieldPayload) ([]byte, error)

	// UnmarshalBitfieldPayload deserializes bytes into a BitfieldPayload
	UnmarshalBitfieldPayload(data []byte, p *protocol.BitfieldPayload) error

	// MarshalHavePayload serializes a HavePayload
	MarshalHavePayload(p *protocol.HavePayload) ([]byte, error)

	// UnmarshalHavePayload deserializes bytes into a HavePayload
	UnmarshalHavePayload(data []byte, p *protocol.HavePayload) error

	// MarshalTransferRequestPayload serializes a TransferRequestPayload
	MarshalTransferRequestPayload(p *protocol.TransferRequestPayload) ([]byte, error)

	// UnmarshalTransferRequestPayload deserializes bytes into a TransferRequestPayload
	UnmarshalTransferRequestPayload(data []byte, p *protocol.TransferRequestPayload) error

	// MarshalTransferPayload serializes a TransferPayload
	MarshalTransferPayload(p *protocol.TransferPayload) ([]byte, error)

	// UnmarshalTransferPayload deserializes bytes into a TransferPayload
	UnmarshalTransferPayload(data []byte, p *protocol.TransferPayload) error
}
