package precond

import (
	"fmt"
)

type PreconditionError struct {
	msg string
}

func (e *PreconditionError) Error() string {
	return e.msg
}

func MustNotBeEmpty(str string, msg ...interface{}) string {
	panicOnError(CheckNotEmpty(str, msg...))
	return str
}

func MustNotBeNil(obj interface{}, msg ...interface{}) interface{} {
	panicOnError(CheckNotNil(obj, msg...))
	return obj
}

func MustBeTrue(b bool, msg ...interface{}) {
	panicOnError(CheckTrue(b, msg...))
}

func CheckNotEmpty(str string, msg ...interface{}) error {
	if str == "" {
		return newError("String must not be empty", msg...)
	}
	return nil
}

func CheckNotNil(obj interface{}, msg ...interface{}) error {
	if obj == nil {
		return newError("Object must not be nil", msg...)
	}
	return nil
}

func CheckTrue(b bool, msg ...interface{}) error {
	if b == false {
		return newError("Expression must be true", msg...)
	}
	return nil
}

func panicOnError(e error) {
	if e != nil {
		panic(e)
	}
}

func newError(defaultMsg string, msg ...interface{}) *PreconditionError {
	return &PreconditionError{msg: newErrMsg(defaultMsg, msg...)}
}

func newErrMsg(defaultMsg string, msg ...interface{}) string {
	if msg != nil {
		switch t := msg[0].(type) {
		case string:
			return fmt.Sprintf(t, msg[1:]...)
		default:
			return fmt.Sprint(msg...)
		}
	}
	return defaultMsg
}
