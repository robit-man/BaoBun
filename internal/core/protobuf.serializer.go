package core

import (
	"fmt"
	"math"

	"github.com/baoswarm/baobun/pkg/protocol"
	pb "github.com/baoswarm/baobun/pkg/protocol/proto"
)

type ProtobufSerializer struct{}

func NewProtobufSerializer() *ProtobufSerializer {
	return &ProtobufSerializer{}
}

func (p *ProtobufSerializer) MarshalPeerMessage(msg *protocol.PeerMessage) ([]byte, error) {
	pbMsg := &pb.PeerMessage{
		InfoHash: msg.InfoHash.Bytes(),
		Type:     p.peerMessageTypeToProto(msg.Type),
		Payload:  msg.Payload,
	}
	return pbMsg.MarshalVT()
}

func (p *ProtobufSerializer) UnmarshalPeerMessage(data []byte, msg *protocol.PeerMessage) error {
	pbMsg := &pb.PeerMessage{}
	if err := pbMsg.UnmarshalVT(data); err != nil {
		return err
	}
	msg.InfoHash = protocol.InfoHash(pbMsg.InfoHash)
	msg.Type = p.protoToPeerMessageType(pbMsg.Type)
	msg.Payload = pbMsg.Payload
	return nil
}

func (p *ProtobufSerializer) MarshalHandshakePayload(pl *protocol.HandshakePayload) ([]byte, error) {
	pbPayload := &pb.HandshakePayload{
		InfoHash: pl.InfoHash.Bytes(),
		PeerId:   pl.PeerID,
	}
	return pbPayload.MarshalVT()
}

func (p *ProtobufSerializer) UnmarshalHandshakePayload(data []byte, pl *protocol.HandshakePayload) error {
	pbPayload := &pb.HandshakePayload{}
	if err := pbPayload.UnmarshalVT(data); err != nil {
		return err
	}
	pl.InfoHash = protocol.InfoHash(pbPayload.InfoHash)
	pl.PeerID = pbPayload.PeerId
	return nil
}

func (p *ProtobufSerializer) MarshalBitfieldPayload(pl *protocol.BitfieldPayload) ([]byte, error) {
	pbPayload := &pb.BitfieldPayload{
		Bits: pl.Bits,
	}
	return pbPayload.MarshalVT()
}

func (p *ProtobufSerializer) UnmarshalBitfieldPayload(data []byte, pl *protocol.BitfieldPayload) error {
	pbPayload := &pb.BitfieldPayload{}
	if err := pbPayload.UnmarshalVT(data); err != nil {
		return err
	}
	pl.Bits = pbPayload.Bits
	return nil
}

func (p *ProtobufSerializer) MarshalHavePayload(pl *protocol.HavePayload) ([]byte, error) {
	pbPayload := &pb.HavePayload{
		UnitIndex: pl.UnitIndex,
	}
	return pbPayload.MarshalVT()
}

func (p *ProtobufSerializer) UnmarshalHavePayload(data []byte, pl *protocol.HavePayload) error {
	pbPayload := &pb.HavePayload{}
	if err := pbPayload.UnmarshalVT(data); err != nil {
		return err
	}
	pl.UnitIndex = pbPayload.UnitIndex
	return nil
}

func (p *ProtobufSerializer) MarshalTransferRequestPayload(pl *protocol.TransferRequestPayload) ([]byte, error) {
	pbPayload := &pb.TransferRequestPayload{
		UnitIndex: pl.UnitIndex,
	}
	return pbPayload.MarshalVT()
}

func (p *ProtobufSerializer) UnmarshalTransferRequestPayload(data []byte, pl *protocol.TransferRequestPayload) error {
	pbPayload := &pb.TransferRequestPayload{}
	if err := pbPayload.UnmarshalVT(data); err != nil {
		return err
	}
	pl.UnitIndex = pbPayload.UnitIndex
	return nil
}

func (p *ProtobufSerializer) MarshalTransferPayload(pl *protocol.TransferPayload) ([]byte, error) {
	pbPayload := &pb.TransferPayload{
		UnitIndex: pl.UnitIndex,
		Data:      pl.Data,
		Proof:     ProofToProto(pl.Proof),
	}
	return pbPayload.MarshalVT()
}

func (p *ProtobufSerializer) UnmarshalTransferPayload(data []byte, pl *protocol.TransferPayload) error {
	pbPayload := &pb.TransferPayload{}
	if err := pbPayload.UnmarshalVT(data); err != nil {
		return err
	}
	pl.UnitIndex = pbPayload.UnitIndex
	pl.Data = pbPayload.Data

	var err error
	pl.Proof, err = ProofFromProto(pbPayload.Proof)
	if err != nil {
		return nil
	}
	return nil
}

func (p *ProtobufSerializer) MarshalRejectPayload(pl *protocol.RejectPayload) ([]byte, error) {
	pbPayload := &pb.RejectPayload{
		UnitIndex: pl.UnitIndex,
		Reason:    pl.Reason,
	}
	return pbPayload.MarshalVT()
}

func (p *ProtobufSerializer) UnmarshalRejectPayload(data []byte, pl *protocol.RejectPayload) error {
	pbPayload := &pb.RejectPayload{}
	if err := pbPayload.UnmarshalVT(data); err != nil {
		return err
	}
	pl.UnitIndex = pbPayload.UnitIndex
	pl.Reason = pbPayload.Reason
	return nil
}

// Helper conversion functions
func (p *ProtobufSerializer) peerMessageTypeToProto(t protocol.PeerMessageType) pb.PeerMessageType {
	switch t {
	case protocol.MsgHandshake:
		return pb.PeerMessageType_MSG_HANDSHAKE
	case protocol.MsgBitfield:
		return pb.PeerMessageType_MSG_BITFIELD
	case protocol.MsgHave:
		return pb.PeerMessageType_MSG_HAVE
	case protocol.MsgRequest:
		return pb.PeerMessageType_MSG_REQUEST
	case protocol.MsgTransfer:
		return pb.PeerMessageType_MSG_TRANSFER
	case protocol.MsgReject:
		return pb.PeerMessageType_MSG_REJECT
	default:
		return pb.PeerMessageType_MSG_HANDSHAKE
	}
}

func (p *ProtobufSerializer) protoToPeerMessageType(t pb.PeerMessageType) protocol.PeerMessageType {
	switch t {
	case pb.PeerMessageType_MSG_HANDSHAKE:
		return protocol.MsgHandshake
	case pb.PeerMessageType_MSG_BITFIELD:
		return protocol.MsgBitfield
	case pb.PeerMessageType_MSG_HAVE:
		return protocol.MsgHave
	case pb.PeerMessageType_MSG_REQUEST:
		return protocol.MsgRequest
	case pb.PeerMessageType_MSG_TRANSFER:
		return protocol.MsgTransfer
	case pb.PeerMessageType_MSG_REJECT:
		return protocol.MsgReject
	default:
		return protocol.MsgHandshake
	}
}

func ProofToProto(p *protocol.Proof) *pb.BaoProof {
	if p == nil {
		return nil
	}

	nodes := make([]*pb.BaoProofNode, len(p.Nodes))
	for i, n := range p.Nodes {
		hash := make([]byte, 32)
		copy(hash, n.Hash[:])

		nodes[i] = &pb.BaoProofNode{
			Hash:  hash,
			Level: uint32(n.Level),
		}
	}

	return &pb.BaoProof{
		LeafStart: p.LeafStart,
		LeafCount: p.LeafCount,
		Proof:     nodes,
	}
}

func ProofFromProto(p *pb.BaoProof) (*protocol.Proof, error) {
	if p == nil {
		return nil, nil
	}

	nodes := make([]protocol.ProofNode, len(p.Proof))
	for i, n := range p.Proof {
		if len(n.Hash) != 32 {
			return nil, fmt.Errorf("invalid hash length: %d", len(n.Hash))
		}

		var hash [32]byte
		copy(hash[:], n.Hash)

		if n.Level > math.MaxUint8 {
			return nil, fmt.Errorf("level out of range: %d", n.Level)
		}

		nodes[i] = protocol.ProofNode{
			Hash:  hash,
			Level: uint8(n.Level),
		}
	}

	return &protocol.Proof{
		LeafStart: p.LeafStart,
		LeafCount: p.LeafCount,
		Nodes:     nodes,
	}, nil
}
