-- name: EditorRequest :one
SELECT * FROM editor_requests WHERE page_id=?;
-- name: EditorCreateRequest :exec
INSERT INTO editor_requests(page_id,id,requester_id,requester_email,requester_avatar_version,requester_session,requester_client,holder_session,holder_client,holder_epoch,deadline,status)
VALUES(?,?,?,?,?,?,?,?,?,?,?,'pending') ON CONFLICT(page_id) DO UPDATE SET id=excluded.id,requester_id=excluded.requester_id,requester_email=excluded.requester_email,requester_avatar_version=excluded.requester_avatar_version,requester_session=excluded.requester_session,requester_client=excluded.requester_client,holder_session=excluded.holder_session,holder_client=excluded.holder_client,holder_epoch=excluded.holder_epoch,deadline=excluded.deadline,status='pending';
-- name: EditorRequestStatus :exec
UPDATE editor_requests SET status=? WHERE page_id=? AND id=?;
-- name: EditorCancelRequests :exec
UPDATE editor_requests SET status='cancelled' WHERE status='pending';

-- name: EditorDeleteUserRequests :exec
DELETE FROM editor_requests WHERE requester_id=? OR page_id IN (SELECT page_id FROM page_leases WHERE holder_user_id=?);

-- name: EditorCancelPageRequest :exec
UPDATE editor_requests SET status='cancelled' WHERE page_id=? AND status='pending';
