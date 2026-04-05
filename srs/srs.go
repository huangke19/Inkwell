package srs

import "time"

const maxInterval = 30

type Result int

const (
	Correct   Result = iota
	Incorrect
)

// Next 根据当前状态和答题结果，返回新的 intervalDays、repetitions 和下次复习的 Unix 时间戳
func Next(currentInterval, repetitions int, result Result) (newInterval, newRepetitions int, nextReviewAt int64) {
	switch result {
	case Correct:
		newRepetitions = repetitions + 1
		if repetitions == 0 {
			newInterval = 1
		} else {
			newInterval = currentInterval * 2
			if newInterval > maxInterval {
				newInterval = maxInterval
			}
		}
	case Incorrect:
		newRepetitions = 0
		newInterval = 1
	}
	nextReviewAt = time.Now().Add(time.Duration(newInterval) * 24 * time.Hour).Unix()
	return
}
