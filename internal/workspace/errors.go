package workspace

import (
	"database/sql"
	"errors"
)

func repositoryError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return notFound()
	}

	return err
}
