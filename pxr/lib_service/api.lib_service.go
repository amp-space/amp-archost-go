package lib_service

import "github.com/arcverse/go-arcverse/pxr"

// LibServiceOpts exposes options and settings
type LibServiceOpts struct {
	ServiceURI string
}

func DefaultLibServiceOpts() LibServiceOpts {
	return LibServiceOpts{
		ServiceURI: "lib",
	}
}

type LibService interface {
	pxr.HostService

	NewLibSession() (LibSession, error)
}

type LibSession interface {
	Close()

	Realloc(buf *[]byte, newLen int64)

	// Blocking calls to send/recv Msgs to the host
	EnqueueIncoming(msg *pxr.Msg) error
	DequeueOutgoing(msg_pb *[]byte) error
}

func (opts LibServiceOpts) NewLibService() LibService {
	return &libService{
		opts: opts,
	}
}