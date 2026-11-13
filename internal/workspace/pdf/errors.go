package pdf

import "mindflow/internal/workspace/storage"

func repositoryError(err error) error {
	if storage.IsNotFound(err) {
		return invalid("This workspace or asset is unavailable.")
	}

	return err
}
