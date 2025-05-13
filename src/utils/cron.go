package utils

import (
	"log"
	"time"

	"github.com/gorhill/cronexpr"
)

func ParseCronSchedule(schedule string) error {
	_, err := cronexpr.Parse(schedule)
	return err
}

func CalculateNextExecution(schedule string, from time.Time) time.Time {
	next := cronexpr.MustParse(schedule).Next(from)
	log.Printf("calculated next schedule %v", next)
	return next
}
