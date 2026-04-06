package models

import (
	"database/sql"
	"strconv"
	"time"
)

type Word struct {
	ID            int64
	Word          string
	Context       string
	SourceURL     string
	SourceTitle   string
	AIMeaning     string // JSON
	AIExamples    string // JSON
	AIScenarios   string // JSON
	AIMemoryTip   string
	AIGeneratedAt int64
	IntervalDays  int
	NextReviewAt  int64
	Repetitions   int
	RatingCEFR    string
	RatingFreq    string
	RatingRec     string
	CreatedAt     int64
	UpdatedAt     int64
}

func (w *Word) AIReady() bool {
	return w.AIMeaning != ""
}

const selectCols = `id,word,context,
	COALESCE(source_url,''),COALESCE(source_title,''),
	COALESCE(ai_meaning,''),COALESCE(ai_examples,''),COALESCE(ai_scenarios,''),COALESCE(ai_memory_tip,''),
	COALESCE(ai_generated_at,0),
	interval_days,next_review_at,repetitions,
	COALESCE(rating_cefr,''),COALESCE(rating_freq,''),COALESCE(rating_rec,''),
	created_at,updated_at`

func CreateWord(db *sql.DB, word, context, sourceURL, sourceTitle string) (*Word, error) {
	res, err := db.Exec(
		`INSERT INTO words (word, context, source_url, source_title) VALUES (?, ?, ?, ?)`,
		word, context, sourceURL, sourceTitle,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return GetWordByID(db, id)
}

func GetWordByID(db *sql.DB, id int64) (*Word, error) {
	row := db.QueryRow(`SELECT `+selectCols+` FROM words WHERE id=?`, id)
	return scanWord(row)
}

func GetWordByText(db *sql.DB, word string) (*Word, error) {
	row := db.QueryRow(`SELECT `+selectCols+` FROM words WHERE word=?`, word)
	return scanWord(row)
}

const freqOrder = `CASE rating_freq WHEN '高频' THEN 0 WHEN '中频' THEN 1 WHEN '低频' THEN 2 ELSE 3 END`
const ratingRecOrder = `CASE rating_rec WHEN '强烈推荐' THEN 0 WHEN '建议掌握' THEN 1 WHEN '选择性记' THEN 2 WHEN '可以跳过' THEN 3 ELSE 4 END`
const cefrOrder = `CASE rating_cefr WHEN 'A1' THEN 0 WHEN 'A2' THEN 1 WHEN 'B1' THEN 2 WHEN 'B2' THEN 3 WHEN 'C1' THEN 4 WHEN 'C2' THEN 5 ELSE 6 END`
const dueFirst = `CASE WHEN next_review_at <= unixepoch() THEN 0 ELSE 1 END`

func wordOrder(sort string) string {
	switch sort {
	case "created":
		return `created_at DESC, ` + ratingRecOrder + ` ASC, ` + freqOrder + ` ASC, ` + cefrOrder + ` ASC, word ASC`
	default:
		return ratingRecOrder + ` ASC, ` + freqOrder + ` ASC, ` + cefrOrder + ` ASC, ` + dueFirst + ` ASC, created_at DESC, word ASC`
	}
}

func ListWords(db *sql.DB, q, sort string, limit, offset int) ([]*Word, error) {
	var rows *sql.Rows
	var err error
	order := wordOrder(sort)
	if q != "" {
		rows, err = db.Query(`SELECT `+selectCols+` FROM words WHERE word LIKE ? ORDER BY `+order+` LIMIT ? OFFSET ?`, "%"+q+"%", limit, offset)
	} else {
		rows, err = db.Query(`SELECT `+selectCols+` FROM words ORDER BY `+order+` LIMIT ? OFFSET ?`, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var words []*Word
	for rows.Next() {
		w := &Word{}
		if err := rows.Scan(&w.ID, &w.Word, &w.Context, &w.SourceURL, &w.SourceTitle,
			&w.AIMeaning, &w.AIExamples, &w.AIScenarios, &w.AIMemoryTip,
			&w.AIGeneratedAt,
			&w.IntervalDays, &w.NextReviewAt, &w.Repetitions,
			&w.RatingCEFR, &w.RatingFreq, &w.RatingRec,
			&w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		words = append(words, w)
	}
	return words, rows.Err()
}

func DeleteWord(db *sql.DB, id int64) error {
	_, err := db.Exec(`DELETE FROM words WHERE id=?`, id)
	return err
}

func UpdateWordAI(db *sql.DB, id int64, meaning, examples, scenarios, memoryTip string) error {
	_, err := db.Exec(`UPDATE words SET
		ai_meaning=?, ai_examples=?, ai_scenarios=?, ai_memory_tip=?,
		ai_generated_at=?, updated_at=unixepoch()
		WHERE id=?`,
		meaning, examples, scenarios, memoryTip, time.Now().Unix(), id)
	return err
}

func UpdateWordRating(db *sql.DB, id int64, cefr, freq, rec string) error {
	_, err := db.Exec(`UPDATE words SET rating_cefr=?, rating_freq=?, rating_rec=?, updated_at=unixepoch() WHERE id=?`,
		cefr, freq, rec, id)
	return err
}

func MarkMastered(db *sql.DB, id int64) error {
	nextAt := time.Now().Add(30 * 24 * time.Hour).Unix()
	_, err := db.Exec(`UPDATE words SET interval_days=30, next_review_at=?, repetitions=6, updated_at=unixepoch() WHERE id=?`,
		nextAt, id)
	return err
}

func UpdateWordSRS(db *sql.DB, id int64, intervalDays int, nextReviewAt int64, repetitions int) error {
	_, err := db.Exec(`UPDATE words SET
		interval_days=?, next_review_at=?, repetitions=?, updated_at=unixepoch()
		WHERE id=?`,
		intervalDays, nextReviewAt, repetitions, id)
	return err
}

func CountDueWords(db *sql.DB) (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM words WHERE next_review_at <= ?`, time.Now().Unix()).Scan(&n)
	return n, err
}

func NextDueWord(db *sql.DB) (*Word, error) {
	row := db.QueryRow(`SELECT `+selectCols+` FROM words WHERE next_review_at <= ? ORDER BY next_review_at ASC LIMIT 1`,
		time.Now().Unix())
	w, err := scanWord(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return w, err
}

func CountWords(db *sql.DB) (total, mastered int, err error) {
	err = db.QueryRow(`SELECT COUNT(*) FROM words`).Scan(&total)
	if err != nil {
		return
	}
	err = db.QueryRow(`SELECT COUNT(*) FROM words WHERE interval_days >= 30`).Scan(&mastered)
	return
}

func CountWordsFiltered(db *sql.DB, q string) (int, error) {
	var total int
	var err error
	if q != "" {
		err = db.QueryRow(`SELECT COUNT(*) FROM words WHERE word LIKE ?`, "%"+q+"%").Scan(&total)
		return total, err
	}
	err = db.QueryRow(`SELECT COUNT(*) FROM words`).Scan(&total)
	return total, err
}

func ListUnmastered(db *sql.DB, q, sort string, limit, offset int) ([]*Word, error) {
	var rows *sql.Rows
	var err error
	order := wordOrder(sort)
	if q != "" {
		rows, err = db.Query(`SELECT `+selectCols+` FROM words WHERE interval_days < 30 AND word LIKE ? ORDER BY `+order+` LIMIT ? OFFSET ?`, "%"+q+"%", limit, offset)
	} else {
		rows, err = db.Query(`SELECT `+selectCols+` FROM words WHERE interval_days < 30 ORDER BY `+order+` LIMIT ? OFFSET ?`, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var words []*Word
	for rows.Next() {
		w := &Word{}
		if err := rows.Scan(&w.ID, &w.Word, &w.Context, &w.SourceURL, &w.SourceTitle,
			&w.AIMeaning, &w.AIExamples, &w.AIScenarios, &w.AIMemoryTip,
			&w.AIGeneratedAt,
			&w.IntervalDays, &w.NextReviewAt, &w.Repetitions,
			&w.RatingCEFR, &w.RatingFreq, &w.RatingRec,
			&w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		words = append(words, w)
	}
	return words, rows.Err()
}

func ListMastered(db *sql.DB, q, sort string, limit, offset int) ([]*Word, error) {
	var rows *sql.Rows
	var err error
	order := wordOrder(sort)
	if q != "" {
		rows, err = db.Query(`SELECT `+selectCols+` FROM words WHERE interval_days >= 30 AND word LIKE ? ORDER BY `+order+` LIMIT ? OFFSET ?`, "%"+q+"%", limit, offset)
	} else {
		rows, err = db.Query(`SELECT `+selectCols+` FROM words WHERE interval_days >= 30 ORDER BY `+order+` LIMIT ? OFFSET ?`, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var words []*Word
	for rows.Next() {
		w := &Word{}
		if err := rows.Scan(&w.ID, &w.Word, &w.Context, &w.SourceURL, &w.SourceTitle,
			&w.AIMeaning, &w.AIExamples, &w.AIScenarios, &w.AIMemoryTip,
			&w.AIGeneratedAt,
			&w.IntervalDays, &w.NextReviewAt, &w.Repetitions,
			&w.RatingCEFR, &w.RatingFreq, &w.RatingRec,
			&w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		words = append(words, w)
	}
	return words, rows.Err()
}

func CountUnmasteredWordsFiltered(db *sql.DB, q string) (int, error) {
	var total int
	var err error
	if q != "" {
		err = db.QueryRow(`SELECT COUNT(*) FROM words WHERE interval_days < 30 AND word LIKE ?`, "%"+q+"%").Scan(&total)
		return total, err
	}
	err = db.QueryRow(`SELECT COUNT(*) FROM words WHERE interval_days < 30`).Scan(&total)
	return total, err
}

func CountMasteredWordsFiltered(db *sql.DB, q string) (int, error) {
	var total int
	var err error
	if q != "" {
		err = db.QueryRow(`SELECT COUNT(*) FROM words WHERE interval_days >= 30 AND word LIKE ?`, "%"+q+"%").Scan(&total)
		return total, err
	}
	err = db.QueryRow(`SELECT COUNT(*) FROM words WHERE interval_days >= 30`).Scan(&total)
	return total, err
}

func FormatDays(days int) string {
	return strconv.Itoa(days) + " 天后"
}

func scanWord(row *sql.Row) (*Word, error) {
	w := &Word{}
	err := row.Scan(&w.ID, &w.Word, &w.Context, &w.SourceURL, &w.SourceTitle,
		&w.AIMeaning, &w.AIExamples, &w.AIScenarios, &w.AIMemoryTip,
		&w.AIGeneratedAt,
		&w.IntervalDays, &w.NextReviewAt, &w.Repetitions,
		&w.RatingCEFR, &w.RatingFreq, &w.RatingRec,
		&w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return w, nil
}
