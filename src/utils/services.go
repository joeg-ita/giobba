package utils

func CheckInterface[T any](s interface{}) bool {
	_, ok := s.(T)
	return ok
}
