package config

import "time"

const (
	DialTimeoutMs          int32 = 10000
	MTU                    int32 = 1024 * 16
	ActiveTransfersPerPeer int   = 32
	ActiveTransfersTotal   int   = 256

	TransferUnitSize       int           = 1024 * 64
	TransferRequestTimeout time.Duration = 60 * time.Second
)
