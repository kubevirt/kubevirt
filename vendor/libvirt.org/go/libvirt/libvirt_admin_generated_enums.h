/*
 * This file is part of the libvirt-go-module project
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
 * Copyright (C) 2022 Red Hat, Inc.
 *
 */
/****************************************************************************
 * THIS CODE HAS BEEN GENERATED. DO NOT CHANGE IT DIRECTLY                  *
 ****************************************************************************/

#pragma once

/* enum virAdmConnectDaemonShutdownFlags */
#  if !LIBVIR_CHECK_VERSION(11, 2, 0)
#    define VIR_DAEMON_SHUTDOWN_PRESERVE (1 << 0)
#  endif

/* enum virClientTransport */
#  if !LIBVIR_CHECK_VERSION(2, 0, 0)
#    define VIR_CLIENT_TRANS_UNIX 0
#  endif
#  if !LIBVIR_CHECK_VERSION(2, 0, 0)
#    define VIR_CLIENT_TRANS_TCP 1
#  endif
#  if !LIBVIR_CHECK_VERSION(2, 0, 0)
#    define VIR_CLIENT_TRANS_TLS 2
#  endif
#  if !LIBVIR_CHECK_VERSION(2, 0, 0)
#    define VIR_CLIENT_TRANS_LAST 3
#  endif

