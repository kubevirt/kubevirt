/*
 * This file is part of the libvirt-go-xml-module project
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
 * Copyright (C) 2017 Red Hat, Inc.
 *
 */

package libvirtxml

type StorageEncryptionSecret struct {
	Type string `xml:"type,attr"`
	UUID string `xml:"uuid,attr"`
}

type StorageEncryptionCipher struct {
	Name string `xml:"name,attr"`
	Size uint64 `xml:"size,attr"`
	Mode string `xml:"mode,attr"`
	Hash string `xml:"hash,attr"`
}

type StorageEncryptionIvgen struct {
	Name string `xml:"name,attr"`
	Hash string `xml:"hash,attr"`
}

type StorageEncryption struct {
	Format string                   `xml:"format,attr"`
	Secret *StorageEncryptionSecret `xml:"secret"`
	Cipher *StorageEncryptionCipher `xml:"cipher"`
	Ivgen  *StorageEncryptionIvgen  `xml:"ivgen"`
}
