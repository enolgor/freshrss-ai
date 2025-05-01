-- name: GetFeeds :many
SELECT * FROM feed;

-- name: GetEntries :many
SELECT * FROM entry WHERE id_feed = sqlc.arg(feed_id) AND date >= sqlc.arg(from_date) AND date < sqlc.arg(to_date) ORDER BY date DESC;

-- name: MarkEntryAsRead :exec
UPDATE entry SET is_read = 1 WHERE id = sqlc.arg(entry_id);