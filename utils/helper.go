package utils

/*
* SetField helper function
* Helps with cleanly unpacking values from *process.Process.func()
 */
func SetField[T any](field *T, getter func() (T, error)) {
	if value, err := getter(); err == nil {
		*field = value
	}
}
