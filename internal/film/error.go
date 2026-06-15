package film

import "errors"

var ErrNotFound = errors.New("Resource not found")
var ErrDuplicateEntry = errors.New("Register already exists")
