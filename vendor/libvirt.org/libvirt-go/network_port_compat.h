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
 * Copyright (C) 2019 Red Hat, Inc.
 *
 */

#ifndef LIBVIRT_GO_NETWORK_PORT_COMPAT_H__
#define LIBVIRT_GO_NETWORK_PORT_COMPAT_H__

/* 5.5.0 */

#if LIBVIR_VERSION_NUMBER < 5005000
typedef struct _virNetworkPort *virNetworkPortPtr;
#endif

#ifndef VIR_NETWORK_PORT_CREATE_RECLAIM
#define VIR_NETWORK_PORT_CREATE_RECLAIM (1 << 0)
#endif

#ifndef VIR_NETWORK_PORT_BANDWIDTH_IN_AVERAGE
#define VIR_NETWORK_PORT_BANDWIDTH_IN_AVERAGE "inbound.average"
#endif

#ifndef VIR_NETWORK_PORT_BANDWIDTH_IN_PEAK
#define VIR_NETWORK_PORT_BANDWIDTH_IN_PEAK "inbound.peak"
#endif

#ifndef VIR_NETWORK_PORT_BANDWIDTH_IN_BURST
#define VIR_NETWORK_PORT_BANDWIDTH_IN_BURST "inbound.burst"
#endif

#ifndef VIR_NETWORK_PORT_BANDWIDTH_IN_FLOOR
#define VIR_NETWORK_PORT_BANDWIDTH_IN_FLOOR "inbound.floor"
#endif

#ifndef VIR_NETWORK_PORT_BANDWIDTH_OUT_AVERAGE
#define VIR_NETWORK_PORT_BANDWIDTH_OUT_AVERAGE "outbound.average"
#endif

#ifndef VIR_NETWORK_PORT_BANDWIDTH_OUT_PEAK
#define VIR_NETWORK_PORT_BANDWIDTH_OUT_PEAK "outbound.peak"
#endif

#ifndef VIR_NETWORK_PORT_BANDWIDTH_OUT_BURST
#define VIR_NETWORK_PORT_BANDWIDTH_OUT_BURST "outbound.burst"
#endif

#endif /* LIBVIRT_GO_NETWORK_PORT_COMPAT_H__ */
