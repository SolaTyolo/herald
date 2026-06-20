package repository

import (
	"errors"

	"github.com/SolaTyolo/herald/internal/repository/filestore"
	"gorm.io/gorm"
)

// IsNotFound reports whether err is a missing-record error from any Store backend.
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, filestore.ErrNotFound)
}
