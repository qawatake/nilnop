package example

import "errors"

func f() (err error) {
	reportError(err) // <- nil is passed to reportError
	err = errors.New("new error")
	reportError(err) // ok because err is not nil
	return err
}

// reportError panics if err is not nil.
func reportError(err error) {
	if err != nil {
		panic(err)
	}
}
