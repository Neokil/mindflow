package canvas

import "mindflow/internal/canvas/storage"

func repositoryError(err error) error {
	if storage.IsNotFound(err) {
		return notFound()
	}

	return err
}
