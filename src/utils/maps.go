package utils

import (
	"iter"
)

func KeysIntoList(keys iter.Seq[string]) []string {
	result := []string{}
	if keys != nil {
		for m := range keys {
			result = append(result, m)
		}
	}
	return result

}
