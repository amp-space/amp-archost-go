package lib_service

import "github.com/amp-3d/amp-sdk-go/amp"

// LibServiceOpts exposes options and settings
type LibServiceOpts struct {
}

func DefaultLibServiceOpts() LibServiceOpts {
	return LibServiceOpts{
	}
}

type LibService interface {
	amp.HostService

	NewLibSession() (LibSession, error)
}

type LibSession interface {
	Close() error

	Realloc(buf *[]byte, newLen int64)

	// Blocking calls to send/recv Msgs to the host
	EnqueueIncoming(tx *amp.TxMsg) error
	DequeueOutgoing(tx_pb *[]byte) error
}

func (opts LibServiceOpts) NewLibService() LibService {
	return &libService{
		opts: opts,
	}
}
