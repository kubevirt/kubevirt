/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

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
