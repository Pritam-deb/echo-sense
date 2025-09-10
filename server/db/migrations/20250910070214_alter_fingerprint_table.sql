-- +goose Up
ALTER TABLE audio_fingerprints DROP COLUMN address;
ALTER TABLE audio_fingerprints DROP COLUMN anchor_time;
ALTER TABLE audio_fingerprints ADD COLUMN hash BIGINT;
ALTER TABLE audio_fingerprints ADD COLUMN anchor_time DOUBLE PRECISION;
CREATE INDEX idx_audiofingerprints_hash ON audio_fingerprints(hash);
CREATE INDEX idx_audiofingerprints_song_id ON audio_fingerprints(song_id);
-- +goose StatementBegin
SELECT 'up SQL query';
-- +goose StatementEnd

-- +goose Down
ALTER TABLE audio_fingerprints DROP COLUMN hash;
ALTER TABLE audio_fingerprints DROP COLUMN anchor_time;
ALTER TABLE audio_fingerprints ADD COLUMN address INT;
ALTER TABLE audio_fingerprints ADD COLUMN anchor_time INT;
DROP INDEX IF EXISTS idx_audiofingerprints_hash;
DROP INDEX IF EXISTS idx_audiofingerprints_song_id;
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
