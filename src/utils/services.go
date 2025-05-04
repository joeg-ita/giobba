package utils

import "github.com/joeg-ita/giobba/src/services"

func CheckInterface(value interface{}) bool {
	_, ok := value.(services.TaskHandlerInt)
	return ok
}
