-- Remove jobs feature flag
DELETE FROM app.settings WHERE key = 'app.features.enable_jobs';
