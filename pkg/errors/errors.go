package errors

import "errors"

var (
	ErrReplicaNotAlive = errors.New("replica not alive")
	ErrorOnSync        = errors.New("failed to sync database from leader")
)
