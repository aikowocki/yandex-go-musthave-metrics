package port

import "context"

type DB interface {
	Ping(ctx context.Context) error
	Close()
}
