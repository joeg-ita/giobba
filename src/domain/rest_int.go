package domain

type RestInt interface {
	Post(url string, payload map[string]interface{}) error
}
