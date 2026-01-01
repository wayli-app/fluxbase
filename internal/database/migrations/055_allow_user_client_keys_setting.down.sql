-- Remove the allow_user_client_keys setting
DELETE FROM app.settings WHERE key = 'app.auth.allow_user_client_keys';
