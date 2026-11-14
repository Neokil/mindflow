-- name: LinkTargetElements :many
SELECT CAST(json_extract(element.value,'$.id') AS TEXT) AS element_id FROM active_pages p,json_each(p.document,'$.elements') element WHERE p.id=sqlc.arg(page_id) AND p.workspace_id=sqlc.arg(workspace_id);
