package storage

type ShortURL struct {
	ID            string `db:"id"`
	LongURL       string `db:"url"`
	UserID        string `db:"user_id"`
	CorrelationID string `db:"correlation_id"`
}
