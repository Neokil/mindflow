-- name: SetupStatus :one
SELECT initialized FROM app_settings WHERE id=1;

-- name: CompleteSetup :execresult
UPDATE app_settings SET initialized=1,config=? WHERE id=1;

-- name: UpdateSettings :execresult
UPDATE app_settings SET config=? WHERE id=1;

-- name: Settings :one
SELECT config FROM app_settings WHERE id=1;
