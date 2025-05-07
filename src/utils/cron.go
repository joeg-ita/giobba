package utils

import (
	"time"

	"github.com/gorhill/cronexpr"
)

func ParseCronSchedule(schedule string) error {
	_, err := cronexpr.Parse(schedule)
	return err
}

func CalculateNextExecution(schedule string, from time.Time) time.Time {
	return cronexpr.MustParse(schedule).Next(from)
}
