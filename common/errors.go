package common

import (
	"errors"
	"fmt"
)

var (
	ErrMissingRelayPubkey = fmt.Errorf("missing relay public key")
	ErrURLEmpty           = errors.New("url is empty")
)
