package a

import "errors"

func f1() {
	err := doSomething()
	if err != nil {
		panic(err)
	}
	Wrap(err) // want "xx"
}

func f2() (err error) {
	Wrap(err) // want "xx"
	err = errors.New("hoge")
	Wrap(err) // ok
	return err
}

func f3() {
	Wrap(nil) // want "xx"
}

func f4(x any) {
	switch x.(type) {
	case nil:
		Wrap(x) // want "xx"
	case int:
		Wrap(x) // ok
	}
}

func doSomething() error {
	return nil
}

func Wrap(x any) {}
