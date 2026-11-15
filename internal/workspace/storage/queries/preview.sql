-- name: PreviewFirstPage :one
SELECT id,revision,CAST(json_array_length(document,'$.elements')=0 AS INTEGER) AS empty
FROM active_pages WHERE workspace_id=? ORDER BY position,id LIMIT 1;

-- name: PreviewCurrent :one
SELECT page_id,revision,content_id,data,empty FROM workspace_previews WHERE workspace_id=?;

-- name: PreviewStore :exec
INSERT INTO workspace_previews(workspace_id,page_id,revision,content_id,data,empty) VALUES(?,?,?,?,?,?)
ON CONFLICT(workspace_id) DO UPDATE SET page_id=excluded.page_id,revision=excluded.revision,content_id=excluded.content_id,data=excluded.data,empty=excluded.empty;
