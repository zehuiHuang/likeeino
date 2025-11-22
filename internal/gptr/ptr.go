package gptr

// 返回指定类型的指针

func Of[T any](v T) *T {
	return &v
}
