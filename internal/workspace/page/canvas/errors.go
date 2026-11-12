package canvas

import "mindflow/internal/workspace/storage"

func repositoryError(err error) error {
	if storage.IsNotFound(err) {
		return notFound()
	}

	return err
}
