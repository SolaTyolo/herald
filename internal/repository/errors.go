package repository

import (
	"errors"

	"github.com/SolaTyolo/herald/internal/repository/filestore"
	"gorm.io/gorm"
)

// ErrNotFound is a missing-record error for callers that are not a Store backend.
var ErrNotFound = errors.New("record not found")

// IsNotFound reports whether err is a missing-record error from any Store backend.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) || errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, filestore.ErrNotFound)
}
