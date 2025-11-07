-- name: GetStats :one
WITH private_chats AS (
    SELECT 
        COUNT(*) as total,
        s.language,
        COUNT(*) as count_by_lang
    FROM chat c
    JOIN settings s ON c.chat_id = s.chat_id
    WHERE c.type = 'private'
        AND c.created_at >= @since_date::TIMESTAMP WITH TIME ZONE
    GROUP BY s.language
),
group_chats AS (
    SELECT 
        COUNT(*) as total,
        s.language,
        COUNT(*) as count_by_lang
    FROM chat c
    JOIN settings s ON c.chat_id = s.chat_id
    WHERE c.type = 'group'
        AND c.created_at >= @since_date::TIMESTAMP WITH TIME ZONE
    GROUP BY s.language
),
downloads_stats AS (
    SELECT 
        COUNT(*) as total_downloads,
        COALESCE(SUM(mf.file_size), 0) as total_size
    FROM media m
    JOIN media_item mi ON mi.media_id = m.id
    JOIN media_format mf ON mf.item_id = mi.id
    WHERE m.created_at >= @since_date::TIMESTAMP WITH TIME ZONE
)
SELECT 
    COALESCE((SELECT SUM(total) FROM private_chats), 0)::BIGINT as total_private_chats,
    COALESCE(
        (SELECT jsonb_object_agg(language, count_by_lang) 
         FROM private_chats),
        '{}'::jsonb
    )::jsonb as private_chats_by_language,
    
    COALESCE((SELECT SUM(total) FROM group_chats), 0)::BIGINT as total_group_chats,
    COALESCE(
        (SELECT jsonb_object_agg(language, count_by_lang) 
         FROM group_chats),
        '{}'::jsonb
    )::jsonb as group_chats_by_language,
    
    COALESCE((SELECT total_downloads FROM downloads_stats), 0)::BIGINT as total_downloads,
    COALESCE((SELECT total_size FROM downloads_stats), 0)::BIGINT as total_downloads_size;