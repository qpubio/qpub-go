// Package instanceid generates time-ordered instance identifiers aligned with qpub-js (UUIDv7).
package instanceid

import (
	"fmt"

	"github.com/google/uuid"
)

// NewUUIDv7 returns a UUID version 7 string.
func NewUUIDv7() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

// NewSocket returns a socket instance ID (socket_<uuidv7>).
func NewSocket() (string, error) {
	u, err := NewUUIDv7()
	if err != nil {
		return "", err
	}
	return "socket_" + u, nil
}

// NewRest returns a REST instance ID (rest_<uuidv7>).
func NewRest() (string, error) {
	u, err := NewUUIDv7()
	if err != nil {
		return "", err
	}
	return "rest_" + u, nil
}

// MustNewSocket panics if UUID generation fails (used at client construction).
func MustNewSocket() string {
	id, err := NewSocket()
	if err != nil {
		panic(fmt.Sprintf("instanceid: %v", err))
	}
	return id
}

// MustNewRest panics if UUID generation fails (used at client construction).
func MustNewRest() string {
	id, err := NewRest()
	if err != nil {
		panic(fmt.Sprintf("instanceid: %v", err))
	}
	return id
}
