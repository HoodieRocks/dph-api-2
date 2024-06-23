package d_errors

import "errors"

var ErrFileTooLarge = errors.New("file too large")
var ErrFileBadExtension = errors.New("file has an invalid extension")
