package platform

import "errors"

// ErrNoAssociation means the OS has no handler for this file type.
var ErrNoAssociation = errors.New("no file association")
