package store

type Storage interface {
	Get(key string) (string, bool)
	Set(key, value string)
	Delete(key string)
}
