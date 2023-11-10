package a

func f() {
	var x error
	Wrap(x) // want "xx"
	var s S
	s.String()
}

func Wrap(x any) {}

type S int

func (s S) String() string {
	return "a"
}
