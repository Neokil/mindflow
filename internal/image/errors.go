package image

import "mindflow/internal/image/storage"

func repositoryError(err error) error {
	if storage.IsNotFound(err) {
		return notFound()
	}

	return err
}
