package handlers

// 这个文件把 models 包的函数包一层，让 handlers 的代码简洁
// 使用小写名避免污染 handlers 的对外接口

import (
	"database/sql"

	"inkwell/models"
)

func createWord(db *sql.DB, word, context, sourceURL, sourceTitle string) (*models.Word, error) {
	return models.CreateWord(db, word, context, sourceURL, sourceTitle)
}

func createWordWithAI(db *sql.DB, word, context, sourceURL, sourceTitle, meaning, examples, scenarios, memoryTip, etymology, collocations, cefr, freq, rec string) (*models.Word, error) {
	return models.CreateWordWithAI(db, word, context, sourceURL, sourceTitle, meaning, examples, scenarios, memoryTip, etymology, collocations, cefr, freq, rec)
}

func getWordByID(db *sql.DB, id int64) (*models.Word, error) {
	return models.GetWordByID(db, id)
}

func getWordByText(db *sql.DB, word string) (*models.Word, error) {
	return models.GetWordByText(db, word)
}

func listWords(db *sql.DB, q, sort string, limit, offset int) ([]*models.Word, error) {
	return models.ListWords(db, q, sort, limit, offset)
}

func deleteWord(db *sql.DB, id int64) error {
	return models.DeleteWord(db, id)
}

func nextDueWord(db *sql.DB) (*models.Word, error) {
	return models.NextDueWord(db)
}

func countDueWords(db *sql.DB) (int, error) {
	return models.CountDueWords(db)
}

func listUnmastered(db *sql.DB, q, sort string, limit, offset int) ([]*models.Word, error) {
	return models.ListUnmastered(db, q, sort, limit, offset)
}

func listMastered(db *sql.DB, q, sort string, limit, offset int) ([]*models.Word, error) {
	return models.ListMastered(db, q, sort, limit, offset)
}

func countUnmasteredWordsFiltered(db *sql.DB, q string) (int, error) {
	return models.CountUnmasteredWordsFiltered(db, q)
}

func countMasteredWordsFiltered(db *sql.DB, q string) (int, error) {
	return models.CountMasteredWordsFiltered(db, q)
}

func markMastered(db *sql.DB, id int64) error {
	return models.MarkMastered(db, id)
}

func wordStats(db *sql.DB) (total, mastered, due int, err error) {
	total, mastered, err = models.CountWords(db)
	if err != nil {
		return
	}
	due, err = models.CountDueWords(db)
	return
}

func countWordsFiltered(db *sql.DB, q string) (int, error) {
	return models.CountWordsFiltered(db, q)
}

func updateWordRating(db *sql.DB, id int64, cefr, freq, rec string) error {
	return models.UpdateWordRating(db, id, cefr, freq, rec)
}

func updateWordSRS(db *sql.DB, id int64, interval int, nextAt int64, reps int) error {
	return models.UpdateWordSRS(db, id, interval, nextAt, reps)
}

func createReviewLog(db *sql.DB, wordID int64, result, answer string, before, after int) error {
	return models.CreateReviewLog(db, wordID, result, answer, before, after)
}
