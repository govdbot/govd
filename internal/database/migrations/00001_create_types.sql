-- +goose Up
-- +goose StatementBegin
CREATE TYPE media_type AS ENUM (
    'photo',
    'video',
    'audio'
);
CREATE TYPE media_codec AS ENUM (
    'avc',
    'hevc',
    'vp9',
    'vp8',
    'av1',
    'webp',
    'aac',
    'opus',
    'vorbis',
    'mp3',
    'flac'
);
CREATE TYPE chat_type AS ENUM (
    'private',
    'group'
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TYPE IF EXISTS media_type;
DROP TYPE IF EXISTS media_codec;
DROP TYPE IF EXISTS chat_type;
-- +goose StatementEnd
