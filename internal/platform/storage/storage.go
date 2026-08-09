// Package storage contains file storage infrastructure.
package storage

import "github.com/L-ka080/DDB-6-case/internal/item"

var _ item.FileStorage = (*LocalStorage)(nil)
var _ item.FileStorage = (*S3Storage)(nil)
