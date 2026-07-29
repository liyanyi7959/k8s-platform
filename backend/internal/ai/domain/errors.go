package domain

import "errors"

var ErrSensitiveResourceExport = errors.New("direct export of sensitive resource is not allowed")
