package internal

type SimpleStruct struct {
	Str     string
	Number  int
	Boolean bool
}

type ComplexStruct struct {
	Arr     []byte
	Nested  SimpleStruct
	Complex []SimpleStruct
	Any     interface{}
}
