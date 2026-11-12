-- name: InstallationSetupStatus :one
SELECT initialized FROM app_settings WHERE id=1;

-- name: InstallationCompleteSetup :execresult
UPDATE app_settings SET initialized=1,config=? WHERE id=1;

-- name: InstallationUpdateSettings :execresult
UPDATE app_settings SET config=? WHERE id=1;

-- name: InstallationSettings :one
SELECT config FROM app_settings WHERE id=1;
