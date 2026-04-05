package models

import "database/sql"

type ReviewLog struct {
	ID             int64
	WordID         int64
	Result         string
	UserAnswer     string
	IntervalBefore int
	IntervalAfter  int
	ReviewedAt     int64
}

func CreateReviewLog(db *sql.DB, wordID int64, result, userAnswer string, intervalBefore, intervalAfter int) error {
	_, err := db.Exec(`INSERT INTO review_logs
		(word_id, result, user_answer, interval_before, interval_after)
		VALUES (?, ?, ?, ?, ?)`,
		wordID, result, userAnswer, intervalBefore, intervalAfter)
	return err
}
