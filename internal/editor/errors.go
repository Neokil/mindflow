package editor

import "mindflow/internal/editor/storage"

func repositoryError(err error) error {
	if storage.IsNotFound(err) {
		return notFound()
	}

	return err
}
