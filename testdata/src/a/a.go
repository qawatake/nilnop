package a

import "errors"

func f1() {
	err := doSomething()
	if err != nil {
		panic(err)
	}
	Wrap(err) // want "nil"
}

func f2() (err error) {
	Wrap(err) // want "nil"
	err = errors.New("hoge")
	Wrap(err) // ok
	return err
}

func f3() {
	Wrap(nil) // want "nil"
}

func f4(x any) {
	switch x.(type) {
	case nil:
		Wrap(x) // want "nil"
	case int:
		Wrap(x) // ok
	}
}

func doSomething() error {
	return nil
}

func Wrap(x any) {}
