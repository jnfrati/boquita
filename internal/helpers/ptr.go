package helpers

func Ptr[I any](v I) *I {
	return &v
}
