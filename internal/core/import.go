package core

import "github.com/baoswarm/baobun/pkg/protocol"

func (c *Client) ImportBao(path string, fileLocation string) (protocol.InfoHash, error) {
	file, err := Load(path)
	if err != nil {
		return protocol.InfoHash{}, err
	}

	ih := protocol.InfoHash(file.InfoHash)

	swarm := NewSwarm(ih, file, fileLocation)

	c.Swarms[ih] = swarm
	c.Sessions.RegisterSwarm(swarm)

	return ih, nil
}

func (c *Client) ImportBaoData(data []byte, fileLocation string) (protocol.InfoHash, error) {
	file, err := LoadFromBytes(data)
	if err != nil {
		return protocol.InfoHash{}, err
	}

	ih := protocol.InfoHash(file.InfoHash)

	swarm := NewSwarm(ih, file, fileLocation)

	c.Swarms[ih] = swarm
	c.Sessions.RegisterSwarm(swarm)

	return ih, nil
}
