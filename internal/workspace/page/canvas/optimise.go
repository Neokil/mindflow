package canvas

import (
	"context"
	"encoding/json"
	"math"
	"mindflow/internal/account/session"
	"time"
)

// OptimiseImagesCommand supplies the screen density and current canvas zoom.
type OptimiseImagesCommand struct {
	Scale float64 `json:"scale"`
}

// OptimiseImagesResult counts newly created display versions.
type OptimiseImagesResult struct {
	Optimised int `json:"optimised"`
	Images    int `json:"images"`
}

// OptimiseImages preserves originals and generates layout-sized display versions.
func (a *Service) OptimiseImages(ctx context.Context, actor session.Principal, pageID string, command OptimiseImagesCommand) (_ OptimiseImagesResult, err error) {
	defer func() { err = repositoryError(err) }()
	if command.Scale == 0 {
		command.Scale = 1
	}
	if math.IsNaN(command.Scale) || math.IsInf(command.Scale, 0) || command.Scale <= 0 || command.Scale > 64 {
		return OptimiseImagesResult{}, invalid("Choose a valid display scale.")
	}
	_, err = a.repository.RequirePage(ctx, pageID, actor.ID, true)
	if err != nil {
		return OptimiseImagesResult{}, err
	}
	snapshot, err := a.GetDocument(ctx, actor, pageID)
	if err != nil {
		return OptimiseImagesResult{}, err
	}
	var document Document
	err = json.Unmarshal(snapshot.Document, &document)
	if err != nil {
		return OptimiseImagesResult{}, wrap("decode saved page", err)
	}
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	targets := imageTargets(document)
	result := OptimiseImagesResult{Images: len(targets)}
	for id, target := range targets {
		created, createErr := a.images.OptimiseVariant(ctx, actor.ID, id, target[0]*command.Scale, target[1]*command.Scale)
		if createErr != nil {
			return result, createErr
		}
		if created {
			result.Optimised++
		}
	}

	return result, nil
}
