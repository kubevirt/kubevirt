/*
 * This file is part of the libvirt-go project
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 * Copyright (c) 2013 Alex Zorin
 * Copyright (C) 2016 Red Hat, Inc.
 *
 */

package libvirt

/*
#cgo pkg-config: libvirt
#include <libvirt/libvirt.h>
#include <libvirt/virterror.h>
#include <stdlib.h>
#include <string.h>
#include "typedparams_wrapper.h"
*/
import "C"

import (
	"fmt"
	"unsafe"
)

type typedParamsFieldInfo struct {
	set *bool
	i   *int
	ui  *uint
	l   *int64
	ul  *uint64
	b   *bool
	d   *float64
	s   *string
	sl  *[]string
}

func typedParamsUnpack(cparams *C.virTypedParameter, cnparams C.int, infomap map[string]typedParamsFieldInfo) (uint, error) {
	count := uint(0)
	for name, value := range infomap {
		var err C.virError
		var ret C.int
		cname := C.CString(name)
		defer C.free(unsafe.Pointer(cname))
		if value.sl != nil {
			for i := 0; i < int(cnparams); i++ {
				var cparam *C.virTypedParameter
				cparam = (*C.virTypedParameter)(unsafe.Pointer(uintptr(unsafe.Pointer(cparams)) +
					(unsafe.Sizeof(*cparam) * uintptr(i))))
				var cs *C.char
				ret = C.virTypedParamsGetStringWrapper(cparam, 1, cname, &cs, &err)
				if ret == 1 {
					*value.sl = append(*value.sl, C.GoString(cs))
					*value.set = true
					count++
				}
			}
		} else {
			if value.i != nil {
				var ci C.int
				ret = C.virTypedParamsGetIntWrapper(cparams, cnparams, cname, &ci, &err)
				if ret == 1 {
					*value.i = int(ci)
				}
			} else if value.ui != nil {
				var cui C.uint
				ret = C.virTypedParamsGetUIntWrapper(cparams, cnparams, cname, &cui, &err)
				if ret == 1 {
					*value.ui = uint(cui)
				}
			} else if value.l != nil {
				var cl C.longlong
				ret = C.virTypedParamsGetLLongWrapper(cparams, cnparams, cname, &cl, &err)
				if ret == 1 {
					*value.l = int64(cl)
				}
			} else if value.ul != nil {
				var cul C.ulonglong
				ret = C.virTypedParamsGetULLongWrapper(cparams, cnparams, cname, &cul, &err)
				if ret == 1 {
					*value.ul = uint64(cul)
				}
			} else if value.d != nil {
				var cd C.double
				ret = C.virTypedParamsGetDoubleWrapper(cparams, cnparams, cname, &cd, &err)
				if ret == 1 {
					*value.d = float64(cd)
				}
			} else if value.b != nil {
				var cb C.int
				ret = C.virTypedParamsGetBooleanWrapper(cparams, cnparams, cname, &cb, &err)
				if ret == 1 {
					if cb == 1 {
						*value.b = true
					} else {
						*value.b = false
					}
				}
			} else if value.s != nil {
				var cs *C.char
				ret = C.virTypedParamsGetStringWrapper(cparams, cnparams, cname, &cs, &err)
				if ret == 1 {
					*value.s = C.GoString(cs)
				}
			}
			if ret == 1 {
				*value.set = true
				count++
			}
		}
	}

	return count, nil
}

func typedParamsNew(nparams C.int) *C.virTypedParameter {
	var cparams *C.virTypedParameter
	memlen := C.size_t(unsafe.Sizeof(*cparams) * uintptr(nparams))
	cparams = (*C.virTypedParameter)(C.malloc(memlen))
	if cparams == nil {
		C.abort()
	}
	C.memset(unsafe.Pointer(cparams), 0, memlen)
	return cparams
}

func typedParamsPackNew(infomap map[string]typedParamsFieldInfo) (*C.virTypedParameter, C.int, error) {
	var cparams C.virTypedParameterPtr
	var nparams C.int
	var maxparams C.int

	defer C.virTypedParamsFree(cparams, nparams)

	for name, value := range infomap {
		if !*value.set {
			continue
		}

		cname := C.CString(name)
		defer C.free(unsafe.Pointer(cname))
		if value.sl != nil {
			/* We're not actually using virTypedParamsAddStringList, as it is
			 * easier to avoid creating a 'char **' in Go to hold all the strings.
			 * We none the less do a version check, because earlier libvirts
			 * would not expect to see multiple string values in a typed params
			 * list with the same field name
			 */
			if C.LIBVIR_VERSION_NUMBER < 1002017 {
				return nil, 0, makeNotImplementedError("virTypedParamsAddStringList")
			}
			for i := 0; i < len(*value.sl); i++ {
				cvalue := C.CString((*value.sl)[i])
				defer C.free(unsafe.Pointer(cvalue))
				var err C.virError
				ret := C.virTypedParamsAddStringWrapper(&cparams, &nparams, &maxparams, cname, cvalue, &err)
				if ret < 0 {
					return nil, 0, makeError(&err)
				}
			}
		} else {
			var err C.virError
			var ret C.int
			if value.i != nil {
				ret = C.virTypedParamsAddIntWrapper(&cparams, &nparams, &maxparams, cname, C.int(*value.i), &err)
			} else if value.ui != nil {
				ret = C.virTypedParamsAddUIntWrapper(&cparams, &nparams, &maxparams, cname, C.uint(*value.ui), &err)
			} else if value.l != nil {
				ret = C.virTypedParamsAddLLongWrapper(&cparams, &nparams, &maxparams, cname, C.longlong(*value.l), &err)
			} else if value.ul != nil {
				ret = C.virTypedParamsAddULLongWrapper(&cparams, &nparams, &maxparams, cname, C.ulonglong(*value.ul), &err)
			} else if value.b != nil {
				v := 0
				if *value.b {
					v = 1
				}
				ret = C.virTypedParamsAddBooleanWrapper(&cparams, &nparams, &maxparams, cname, C.int(v), &err)
			} else if value.d != nil {
				ret = C.virTypedParamsAddDoubleWrapper(&cparams, &nparams, &maxparams, cname, C.double(*value.i), &err)
			} else if value.s != nil {
				cvalue := C.CString(*value.s)
				defer C.free(unsafe.Pointer(cvalue))
				ret = C.virTypedParamsAddStringWrapper(&cparams, &nparams, &maxparams, cname, cvalue, &err)
			} else {
				return nil, 0, fmt.Errorf("No typed parameter value set for field '%s'", name)
			}
			if ret < 0 {
				return nil, 0, makeError(&err)
			}
		}
	}

	return cparams, nparams, nil
}
