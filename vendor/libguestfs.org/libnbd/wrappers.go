/* NBD client library in userspace
 * WARNING: THIS FILE IS GENERATED FROM
 * generator/generator
 * ANY CHANGES YOU MAKE TO THIS FILE WILL BE LOST.
 *
 * Copyright Red Hat
 *
 * This library is free software; you can redistribute it and/or
 * modify it under the terms of the GNU Lesser General Public
 * License as published by the Free Software Foundation; either
 * version 2 of the License, or (at your option) any later version.
 *
 * This library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 * Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public
 * License along with this library; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA
 */

package libnbd

/*
#cgo pkg-config: libnbd
#cgo CFLAGS: -D_GNU_SOURCE=1

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "libnbd.h"
#include "wrappers.h"

int
_nbd_set_debug_wrapper (struct error *err,
        struct nbd_handle *h, bool debug)
{
#ifdef LIBNBD_HAVE_NBD_SET_DEBUG
  int ret;

  ret = nbd_set_debug (h, debug);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_DEBUG
  missing_function (err, "set_debug");
  return -1;
#endif
}

int
_nbd_get_debug_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_DEBUG
  int ret;

  ret = nbd_get_debug (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_DEBUG
  missing_function (err, "get_debug");
  return -1;
#endif
}

int
_nbd_set_debug_callback_wrapper (struct error *err,
        struct nbd_handle *h, nbd_debug_callback debug_callback)
{
#ifdef LIBNBD_HAVE_NBD_SET_DEBUG_CALLBACK
  int ret;

  ret = nbd_set_debug_callback (h, debug_callback);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_DEBUG_CALLBACK
  missing_function (err, "set_debug_callback");
  return -1;
#endif
}

int
_nbd_clear_debug_callback_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_CLEAR_DEBUG_CALLBACK
  int ret;

  ret = nbd_clear_debug_callback (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_CLEAR_DEBUG_CALLBACK
  missing_function (err, "clear_debug_callback");
  return -1;
#endif
}

uint64_t
_nbd_stats_bytes_sent_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_STATS_BYTES_SENT
  uint64_t ret;

  ret = nbd_stats_bytes_sent (h);
  return ret;
#else // !LIBNBD_HAVE_NBD_STATS_BYTES_SENT
  missing_function (err, "stats_bytes_sent");
#endif
}

uint64_t
_nbd_stats_chunks_sent_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_STATS_CHUNKS_SENT
  uint64_t ret;

  ret = nbd_stats_chunks_sent (h);
  return ret;
#else // !LIBNBD_HAVE_NBD_STATS_CHUNKS_SENT
  missing_function (err, "stats_chunks_sent");
#endif
}

uint64_t
_nbd_stats_bytes_received_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_STATS_BYTES_RECEIVED
  uint64_t ret;

  ret = nbd_stats_bytes_received (h);
  return ret;
#else // !LIBNBD_HAVE_NBD_STATS_BYTES_RECEIVED
  missing_function (err, "stats_bytes_received");
#endif
}

uint64_t
_nbd_stats_chunks_received_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_STATS_CHUNKS_RECEIVED
  uint64_t ret;

  ret = nbd_stats_chunks_received (h);
  return ret;
#else // !LIBNBD_HAVE_NBD_STATS_CHUNKS_RECEIVED
  missing_function (err, "stats_chunks_received");
#endif
}

int
_nbd_set_handle_name_wrapper (struct error *err,
        struct nbd_handle *h, const char *handle_name)
{
#ifdef LIBNBD_HAVE_NBD_SET_HANDLE_NAME
  int ret;

  ret = nbd_set_handle_name (h, handle_name);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_HANDLE_NAME
  missing_function (err, "set_handle_name");
  return -1;
#endif
}

char *
_nbd_get_handle_name_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_HANDLE_NAME
  char * ret;

  ret = nbd_get_handle_name (h);
  if (ret == NULL)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_HANDLE_NAME
  missing_function (err, "get_handle_name");
  return NULL;
#endif
}

uintptr_t
_nbd_set_private_data_wrapper (struct error *err,
        struct nbd_handle *h, uintptr_t private_data)
{
#ifdef LIBNBD_HAVE_NBD_SET_PRIVATE_DATA
  uintptr_t ret;

  ret = nbd_set_private_data (h, private_data);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_PRIVATE_DATA
  missing_function (err, "set_private_data");
#endif
}

uintptr_t
_nbd_get_private_data_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_PRIVATE_DATA
  uintptr_t ret;

  ret = nbd_get_private_data (h);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_PRIVATE_DATA
  missing_function (err, "get_private_data");
#endif
}

int
_nbd_set_export_name_wrapper (struct error *err,
        struct nbd_handle *h, const char *export_name)
{
#ifdef LIBNBD_HAVE_NBD_SET_EXPORT_NAME
  int ret;

  ret = nbd_set_export_name (h, export_name);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_EXPORT_NAME
  missing_function (err, "set_export_name");
  return -1;
#endif
}

char *
_nbd_get_export_name_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_EXPORT_NAME
  char * ret;

  ret = nbd_get_export_name (h);
  if (ret == NULL)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_EXPORT_NAME
  missing_function (err, "get_export_name");
  return NULL;
#endif
}

int
_nbd_set_request_block_size_wrapper (struct error *err,
        struct nbd_handle *h, bool request)
{
#ifdef LIBNBD_HAVE_NBD_SET_REQUEST_BLOCK_SIZE
  int ret;

  ret = nbd_set_request_block_size (h, request);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_REQUEST_BLOCK_SIZE
  missing_function (err, "set_request_block_size");
  return -1;
#endif
}

int
_nbd_get_request_block_size_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_REQUEST_BLOCK_SIZE
  int ret;

  ret = nbd_get_request_block_size (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_REQUEST_BLOCK_SIZE
  missing_function (err, "get_request_block_size");
  return -1;
#endif
}

int
_nbd_set_full_info_wrapper (struct error *err,
        struct nbd_handle *h, bool request)
{
#ifdef LIBNBD_HAVE_NBD_SET_FULL_INFO
  int ret;

  ret = nbd_set_full_info (h, request);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_FULL_INFO
  missing_function (err, "set_full_info");
  return -1;
#endif
}

int
_nbd_get_full_info_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_FULL_INFO
  int ret;

  ret = nbd_get_full_info (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_FULL_INFO
  missing_function (err, "get_full_info");
  return -1;
#endif
}

char *
_nbd_get_canonical_export_name_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_CANONICAL_EXPORT_NAME
  char * ret;

  ret = nbd_get_canonical_export_name (h);
  if (ret == NULL)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_CANONICAL_EXPORT_NAME
  missing_function (err, "get_canonical_export_name");
  return NULL;
#endif
}

char *
_nbd_get_export_description_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_EXPORT_DESCRIPTION
  char * ret;

  ret = nbd_get_export_description (h);
  if (ret == NULL)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_EXPORT_DESCRIPTION
  missing_function (err, "get_export_description");
  return NULL;
#endif
}

int
_nbd_set_tls_wrapper (struct error *err,
        struct nbd_handle *h, int tls)
{
#ifdef LIBNBD_HAVE_NBD_SET_TLS
  int ret;

  ret = nbd_set_tls (h, tls);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_TLS
  missing_function (err, "set_tls");
  return -1;
#endif
}

int
_nbd_get_tls_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_TLS
  int ret;

  ret = nbd_get_tls (h);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_TLS
  missing_function (err, "get_tls");
#endif
}

int
_nbd_get_tls_negotiated_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_TLS_NEGOTIATED
  int ret;

  ret = nbd_get_tls_negotiated (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_TLS_NEGOTIATED
  missing_function (err, "get_tls_negotiated");
  return -1;
#endif
}

int
_nbd_set_tls_certificates_wrapper (struct error *err,
        struct nbd_handle *h, const char *dir)
{
#ifdef LIBNBD_HAVE_NBD_SET_TLS_CERTIFICATES
  int ret;

  ret = nbd_set_tls_certificates (h, dir);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_TLS_CERTIFICATES
  missing_function (err, "set_tls_certificates");
  return -1;
#endif
}

int
_nbd_set_tls_verify_peer_wrapper (struct error *err,
        struct nbd_handle *h, bool verify)
{
#ifdef LIBNBD_HAVE_NBD_SET_TLS_VERIFY_PEER
  int ret;

  ret = nbd_set_tls_verify_peer (h, verify);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_TLS_VERIFY_PEER
  missing_function (err, "set_tls_verify_peer");
  return -1;
#endif
}

int
_nbd_get_tls_verify_peer_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_TLS_VERIFY_PEER
  int ret;

  ret = nbd_get_tls_verify_peer (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_TLS_VERIFY_PEER
  missing_function (err, "get_tls_verify_peer");
  return -1;
#endif
}

int
_nbd_set_tls_username_wrapper (struct error *err,
        struct nbd_handle *h, const char *username)
{
#ifdef LIBNBD_HAVE_NBD_SET_TLS_USERNAME
  int ret;

  ret = nbd_set_tls_username (h, username);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_TLS_USERNAME
  missing_function (err, "set_tls_username");
  return -1;
#endif
}

char *
_nbd_get_tls_username_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_TLS_USERNAME
  char * ret;

  ret = nbd_get_tls_username (h);
  if (ret == NULL)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_TLS_USERNAME
  missing_function (err, "get_tls_username");
  return NULL;
#endif
}

int
_nbd_set_tls_psk_file_wrapper (struct error *err,
        struct nbd_handle *h, const char *filename)
{
#ifdef LIBNBD_HAVE_NBD_SET_TLS_PSK_FILE
  int ret;

  ret = nbd_set_tls_psk_file (h, filename);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_TLS_PSK_FILE
  missing_function (err, "set_tls_psk_file");
  return -1;
#endif
}

int
_nbd_set_request_extended_headers_wrapper (struct error *err,
        struct nbd_handle *h, bool request)
{
#ifdef LIBNBD_HAVE_NBD_SET_REQUEST_EXTENDED_HEADERS
  int ret;

  ret = nbd_set_request_extended_headers (h, request);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_REQUEST_EXTENDED_HEADERS
  missing_function (err, "set_request_extended_headers");
  return -1;
#endif
}

int
_nbd_get_request_extended_headers_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_REQUEST_EXTENDED_HEADERS
  int ret;

  ret = nbd_get_request_extended_headers (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_REQUEST_EXTENDED_HEADERS
  missing_function (err, "get_request_extended_headers");
  return -1;
#endif
}

int
_nbd_get_extended_headers_negotiated_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_EXTENDED_HEADERS_NEGOTIATED
  int ret;

  ret = nbd_get_extended_headers_negotiated (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_EXTENDED_HEADERS_NEGOTIATED
  missing_function (err, "get_extended_headers_negotiated");
  return -1;
#endif
}

int
_nbd_set_request_structured_replies_wrapper (struct error *err,
        struct nbd_handle *h, bool request)
{
#ifdef LIBNBD_HAVE_NBD_SET_REQUEST_STRUCTURED_REPLIES
  int ret;

  ret = nbd_set_request_structured_replies (h, request);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_REQUEST_STRUCTURED_REPLIES
  missing_function (err, "set_request_structured_replies");
  return -1;
#endif
}

int
_nbd_get_request_structured_replies_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_REQUEST_STRUCTURED_REPLIES
  int ret;

  ret = nbd_get_request_structured_replies (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_REQUEST_STRUCTURED_REPLIES
  missing_function (err, "get_request_structured_replies");
  return -1;
#endif
}

int
_nbd_get_structured_replies_negotiated_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_STRUCTURED_REPLIES_NEGOTIATED
  int ret;

  ret = nbd_get_structured_replies_negotiated (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_STRUCTURED_REPLIES_NEGOTIATED
  missing_function (err, "get_structured_replies_negotiated");
  return -1;
#endif
}

int
_nbd_set_request_meta_context_wrapper (struct error *err,
        struct nbd_handle *h, bool request)
{
#ifdef LIBNBD_HAVE_NBD_SET_REQUEST_META_CONTEXT
  int ret;

  ret = nbd_set_request_meta_context (h, request);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_REQUEST_META_CONTEXT
  missing_function (err, "set_request_meta_context");
  return -1;
#endif
}

int
_nbd_get_request_meta_context_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_REQUEST_META_CONTEXT
  int ret;

  ret = nbd_get_request_meta_context (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_REQUEST_META_CONTEXT
  missing_function (err, "get_request_meta_context");
  return -1;
#endif
}

int
_nbd_set_handshake_flags_wrapper (struct error *err,
        struct nbd_handle *h, uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_SET_HANDSHAKE_FLAGS
  int ret;

  ret = nbd_set_handshake_flags (h, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_HANDSHAKE_FLAGS
  missing_function (err, "set_handshake_flags");
  return -1;
#endif
}

uint32_t
_nbd_get_handshake_flags_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_HANDSHAKE_FLAGS
  uint32_t ret;

  ret = nbd_get_handshake_flags (h);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_HANDSHAKE_FLAGS
  missing_function (err, "get_handshake_flags");
#endif
}

int
_nbd_set_pread_initialize_wrapper (struct error *err,
        struct nbd_handle *h, bool request)
{
#ifdef LIBNBD_HAVE_NBD_SET_PREAD_INITIALIZE
  int ret;

  ret = nbd_set_pread_initialize (h, request);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_PREAD_INITIALIZE
  missing_function (err, "set_pread_initialize");
  return -1;
#endif
}

int
_nbd_get_pread_initialize_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_PREAD_INITIALIZE
  int ret;

  ret = nbd_get_pread_initialize (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_PREAD_INITIALIZE
  missing_function (err, "get_pread_initialize");
  return -1;
#endif
}

int
_nbd_set_strict_mode_wrapper (struct error *err,
        struct nbd_handle *h, uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_SET_STRICT_MODE
  int ret;

  ret = nbd_set_strict_mode (h, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_STRICT_MODE
  missing_function (err, "set_strict_mode");
  return -1;
#endif
}

uint32_t
_nbd_get_strict_mode_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_STRICT_MODE
  uint32_t ret;

  ret = nbd_get_strict_mode (h);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_STRICT_MODE
  missing_function (err, "get_strict_mode");
#endif
}

int
_nbd_set_opt_mode_wrapper (struct error *err,
        struct nbd_handle *h, bool enable)
{
#ifdef LIBNBD_HAVE_NBD_SET_OPT_MODE
  int ret;

  ret = nbd_set_opt_mode (h, enable);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_OPT_MODE
  missing_function (err, "set_opt_mode");
  return -1;
#endif
}

int
_nbd_get_opt_mode_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_OPT_MODE
  int ret;

  ret = nbd_get_opt_mode (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_OPT_MODE
  missing_function (err, "get_opt_mode");
  return -1;
#endif
}

int
_nbd_opt_go_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_OPT_GO
  int ret;

  ret = nbd_opt_go (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_OPT_GO
  missing_function (err, "opt_go");
  return -1;
#endif
}

int
_nbd_opt_abort_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_OPT_ABORT
  int ret;

  ret = nbd_opt_abort (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_OPT_ABORT
  missing_function (err, "opt_abort");
  return -1;
#endif
}

int
_nbd_opt_starttls_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_OPT_STARTTLS
  int ret;

  ret = nbd_opt_starttls (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_OPT_STARTTLS
  missing_function (err, "opt_starttls");
  return -1;
#endif
}

int
_nbd_opt_extended_headers_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_OPT_EXTENDED_HEADERS
  int ret;

  ret = nbd_opt_extended_headers (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_OPT_EXTENDED_HEADERS
  missing_function (err, "opt_extended_headers");
  return -1;
#endif
}

int
_nbd_opt_structured_reply_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_OPT_STRUCTURED_REPLY
  int ret;

  ret = nbd_opt_structured_reply (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_OPT_STRUCTURED_REPLY
  missing_function (err, "opt_structured_reply");
  return -1;
#endif
}

int
_nbd_opt_list_wrapper (struct error *err,
        struct nbd_handle *h, nbd_list_callback list_callback)
{
#ifdef LIBNBD_HAVE_NBD_OPT_LIST
  int ret;

  ret = nbd_opt_list (h, list_callback);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_OPT_LIST
  missing_function (err, "opt_list");
  return -1;
#endif
}

int
_nbd_opt_info_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_OPT_INFO
  int ret;

  ret = nbd_opt_info (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_OPT_INFO
  missing_function (err, "opt_info");
  return -1;
#endif
}

int
_nbd_opt_list_meta_context_wrapper (struct error *err,
        struct nbd_handle *h, nbd_context_callback context_callback)
{
#ifdef LIBNBD_HAVE_NBD_OPT_LIST_META_CONTEXT
  int ret;

  ret = nbd_opt_list_meta_context (h, context_callback);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_OPT_LIST_META_CONTEXT
  missing_function (err, "opt_list_meta_context");
  return -1;
#endif
}

int
_nbd_opt_list_meta_context_queries_wrapper (struct error *err,
        struct nbd_handle *h, char **queries,
        nbd_context_callback context_callback)
{
#ifdef LIBNBD_HAVE_NBD_OPT_LIST_META_CONTEXT_QUERIES
  int ret;

  ret = nbd_opt_list_meta_context_queries (h, queries, context_callback);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_OPT_LIST_META_CONTEXT_QUERIES
  missing_function (err, "opt_list_meta_context_queries");
  return -1;
#endif
}

int
_nbd_opt_set_meta_context_wrapper (struct error *err,
        struct nbd_handle *h, nbd_context_callback context_callback)
{
#ifdef LIBNBD_HAVE_NBD_OPT_SET_META_CONTEXT
  int ret;

  ret = nbd_opt_set_meta_context (h, context_callback);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_OPT_SET_META_CONTEXT
  missing_function (err, "opt_set_meta_context");
  return -1;
#endif
}

int
_nbd_opt_set_meta_context_queries_wrapper (struct error *err,
        struct nbd_handle *h, char **queries,
        nbd_context_callback context_callback)
{
#ifdef LIBNBD_HAVE_NBD_OPT_SET_META_CONTEXT_QUERIES
  int ret;

  ret = nbd_opt_set_meta_context_queries (h, queries, context_callback);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_OPT_SET_META_CONTEXT_QUERIES
  missing_function (err, "opt_set_meta_context_queries");
  return -1;
#endif
}

int
_nbd_add_meta_context_wrapper (struct error *err,
        struct nbd_handle *h, const char *name)
{
#ifdef LIBNBD_HAVE_NBD_ADD_META_CONTEXT
  int ret;

  ret = nbd_add_meta_context (h, name);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_ADD_META_CONTEXT
  missing_function (err, "add_meta_context");
  return -1;
#endif
}

ssize_t
_nbd_get_nr_meta_contexts_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_NR_META_CONTEXTS
  ssize_t ret;

  ret = nbd_get_nr_meta_contexts (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_NR_META_CONTEXTS
  missing_function (err, "get_nr_meta_contexts");
  return -1;
#endif
}

char *
_nbd_get_meta_context_wrapper (struct error *err,
        struct nbd_handle *h, size_t i)
{
#ifdef LIBNBD_HAVE_NBD_GET_META_CONTEXT
  char * ret;

  ret = nbd_get_meta_context (h, i);
  if (ret == NULL)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_META_CONTEXT
  missing_function (err, "get_meta_context");
  return NULL;
#endif
}

int
_nbd_clear_meta_contexts_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_CLEAR_META_CONTEXTS
  int ret;

  ret = nbd_clear_meta_contexts (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_CLEAR_META_CONTEXTS
  missing_function (err, "clear_meta_contexts");
  return -1;
#endif
}

int
_nbd_set_uri_allow_transports_wrapper (struct error *err,
        struct nbd_handle *h, uint32_t mask)
{
#ifdef LIBNBD_HAVE_NBD_SET_URI_ALLOW_TRANSPORTS
  int ret;

  ret = nbd_set_uri_allow_transports (h, mask);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_URI_ALLOW_TRANSPORTS
  missing_function (err, "set_uri_allow_transports");
  return -1;
#endif
}

int
_nbd_set_uri_allow_tls_wrapper (struct error *err,
        struct nbd_handle *h, int tls)
{
#ifdef LIBNBD_HAVE_NBD_SET_URI_ALLOW_TLS
  int ret;

  ret = nbd_set_uri_allow_tls (h, tls);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_URI_ALLOW_TLS
  missing_function (err, "set_uri_allow_tls");
  return -1;
#endif
}

int
_nbd_set_uri_allow_local_file_wrapper (struct error *err,
        struct nbd_handle *h, bool allow)
{
#ifdef LIBNBD_HAVE_NBD_SET_URI_ALLOW_LOCAL_FILE
  int ret;

  ret = nbd_set_uri_allow_local_file (h, allow);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_URI_ALLOW_LOCAL_FILE
  missing_function (err, "set_uri_allow_local_file");
  return -1;
#endif
}

int
_nbd_connect_uri_wrapper (struct error *err,
        struct nbd_handle *h, const char *uri)
{
#ifdef LIBNBD_HAVE_NBD_CONNECT_URI
  int ret;

  ret = nbd_connect_uri (h, uri);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_CONNECT_URI
  missing_function (err, "connect_uri");
  return -1;
#endif
}

int
_nbd_connect_unix_wrapper (struct error *err,
        struct nbd_handle *h, const char *unixsocket)
{
#ifdef LIBNBD_HAVE_NBD_CONNECT_UNIX
  int ret;

  ret = nbd_connect_unix (h, unixsocket);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_CONNECT_UNIX
  missing_function (err, "connect_unix");
  return -1;
#endif
}

int
_nbd_connect_vsock_wrapper (struct error *err,
        struct nbd_handle *h, uint32_t cid, uint32_t port)
{
#ifdef LIBNBD_HAVE_NBD_CONNECT_VSOCK
  int ret;

  ret = nbd_connect_vsock (h, cid, port);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_CONNECT_VSOCK
  missing_function (err, "connect_vsock");
  return -1;
#endif
}

int
_nbd_connect_tcp_wrapper (struct error *err,
        struct nbd_handle *h, const char *hostname, const char *port)
{
#ifdef LIBNBD_HAVE_NBD_CONNECT_TCP
  int ret;

  ret = nbd_connect_tcp (h, hostname, port);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_CONNECT_TCP
  missing_function (err, "connect_tcp");
  return -1;
#endif
}

int
_nbd_connect_socket_wrapper (struct error *err,
        struct nbd_handle *h, int sock)
{
#ifdef LIBNBD_HAVE_NBD_CONNECT_SOCKET
  int ret;

  ret = nbd_connect_socket (h, sock);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_CONNECT_SOCKET
  missing_function (err, "connect_socket");
  return -1;
#endif
}

int
_nbd_connect_command_wrapper (struct error *err,
        struct nbd_handle *h, char **argv)
{
#ifdef LIBNBD_HAVE_NBD_CONNECT_COMMAND
  int ret;

  ret = nbd_connect_command (h, argv);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_CONNECT_COMMAND
  missing_function (err, "connect_command");
  return -1;
#endif
}

int
_nbd_connect_systemd_socket_activation_wrapper (struct error *err,
        struct nbd_handle *h, char **argv)
{
#ifdef LIBNBD_HAVE_NBD_CONNECT_SYSTEMD_SOCKET_ACTIVATION
  int ret;

  ret = nbd_connect_systemd_socket_activation (h, argv);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_CONNECT_SYSTEMD_SOCKET_ACTIVATION
  missing_function (err, "connect_systemd_socket_activation");
  return -1;
#endif
}

int
_nbd_set_socket_activation_name_wrapper (struct error *err,
        struct nbd_handle *h, const char *socket_name)
{
#ifdef LIBNBD_HAVE_NBD_SET_SOCKET_ACTIVATION_NAME
  int ret;

  ret = nbd_set_socket_activation_name (h, socket_name);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SET_SOCKET_ACTIVATION_NAME
  missing_function (err, "set_socket_activation_name");
  return -1;
#endif
}

char *
_nbd_get_socket_activation_name_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_SOCKET_ACTIVATION_NAME
  char * ret;

  ret = nbd_get_socket_activation_name (h);
  if (ret == NULL)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_SOCKET_ACTIVATION_NAME
  missing_function (err, "get_socket_activation_name");
  return NULL;
#endif
}

int
_nbd_is_read_only_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_IS_READ_ONLY
  int ret;

  ret = nbd_is_read_only (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_IS_READ_ONLY
  missing_function (err, "is_read_only");
  return -1;
#endif
}

int
_nbd_can_flush_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_CAN_FLUSH
  int ret;

  ret = nbd_can_flush (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_CAN_FLUSH
  missing_function (err, "can_flush");
  return -1;
#endif
}

int
_nbd_can_fua_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_CAN_FUA
  int ret;

  ret = nbd_can_fua (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_CAN_FUA
  missing_function (err, "can_fua");
  return -1;
#endif
}

int
_nbd_is_rotational_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_IS_ROTATIONAL
  int ret;

  ret = nbd_is_rotational (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_IS_ROTATIONAL
  missing_function (err, "is_rotational");
  return -1;
#endif
}

int
_nbd_can_trim_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_CAN_TRIM
  int ret;

  ret = nbd_can_trim (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_CAN_TRIM
  missing_function (err, "can_trim");
  return -1;
#endif
}

int
_nbd_can_zero_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_CAN_ZERO
  int ret;

  ret = nbd_can_zero (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_CAN_ZERO
  missing_function (err, "can_zero");
  return -1;
#endif
}

int
_nbd_can_fast_zero_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_CAN_FAST_ZERO
  int ret;

  ret = nbd_can_fast_zero (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_CAN_FAST_ZERO
  missing_function (err, "can_fast_zero");
  return -1;
#endif
}

int
_nbd_can_block_status_payload_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_CAN_BLOCK_STATUS_PAYLOAD
  int ret;

  ret = nbd_can_block_status_payload (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_CAN_BLOCK_STATUS_PAYLOAD
  missing_function (err, "can_block_status_payload");
  return -1;
#endif
}

int
_nbd_can_df_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_CAN_DF
  int ret;

  ret = nbd_can_df (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_CAN_DF
  missing_function (err, "can_df");
  return -1;
#endif
}

int
_nbd_can_multi_conn_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_CAN_MULTI_CONN
  int ret;

  ret = nbd_can_multi_conn (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_CAN_MULTI_CONN
  missing_function (err, "can_multi_conn");
  return -1;
#endif
}

int
_nbd_can_cache_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_CAN_CACHE
  int ret;

  ret = nbd_can_cache (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_CAN_CACHE
  missing_function (err, "can_cache");
  return -1;
#endif
}

int
_nbd_can_meta_context_wrapper (struct error *err,
        struct nbd_handle *h, const char *metacontext)
{
#ifdef LIBNBD_HAVE_NBD_CAN_META_CONTEXT
  int ret;

  ret = nbd_can_meta_context (h, metacontext);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_CAN_META_CONTEXT
  missing_function (err, "can_meta_context");
  return -1;
#endif
}

const char *
_nbd_get_protocol_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_PROTOCOL
  const char * ret;

  ret = nbd_get_protocol (h);
  if (ret == NULL)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_PROTOCOL
  missing_function (err, "get_protocol");
  return NULL;
#endif
}

int64_t
_nbd_get_size_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_SIZE
  int64_t ret;

  ret = nbd_get_size (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_SIZE
  missing_function (err, "get_size");
  return -1;
#endif
}

int64_t
_nbd_get_block_size_wrapper (struct error *err,
        struct nbd_handle *h, int size_type)
{
#ifdef LIBNBD_HAVE_NBD_GET_BLOCK_SIZE
  int64_t ret;

  ret = nbd_get_block_size (h, size_type);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_BLOCK_SIZE
  missing_function (err, "get_block_size");
  return -1;
#endif
}

int
_nbd_pread_wrapper (struct error *err,
        struct nbd_handle *h, void *buf, size_t count, uint64_t offset,
        uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_PREAD
  int ret;

  ret = nbd_pread (h, buf, count, offset, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_PREAD
  missing_function (err, "pread");
  return -1;
#endif
}

int
_nbd_pread_structured_wrapper (struct error *err,
        struct nbd_handle *h, void *buf, size_t count, uint64_t offset,
        nbd_chunk_callback chunk_callback, uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_PREAD_STRUCTURED
  int ret;

  ret = nbd_pread_structured (h, buf, count, offset, chunk_callback, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_PREAD_STRUCTURED
  missing_function (err, "pread_structured");
  return -1;
#endif
}

int
_nbd_pwrite_wrapper (struct error *err,
        struct nbd_handle *h, const void *buf, size_t count,
        uint64_t offset, uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_PWRITE
  int ret;

  ret = nbd_pwrite (h, buf, count, offset, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_PWRITE
  missing_function (err, "pwrite");
  return -1;
#endif
}

int
_nbd_shutdown_wrapper (struct error *err,
        struct nbd_handle *h, uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_SHUTDOWN
  int ret;

  ret = nbd_shutdown (h, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SHUTDOWN
  missing_function (err, "shutdown");
  return -1;
#endif
}

int
_nbd_flush_wrapper (struct error *err,
        struct nbd_handle *h, uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_FLUSH
  int ret;

  ret = nbd_flush (h, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_FLUSH
  missing_function (err, "flush");
  return -1;
#endif
}

int
_nbd_trim_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_TRIM
  int ret;

  ret = nbd_trim (h, count, offset, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_TRIM
  missing_function (err, "trim");
  return -1;
#endif
}

int
_nbd_cache_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_CACHE
  int ret;

  ret = nbd_cache (h, count, offset, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_CACHE
  missing_function (err, "cache");
  return -1;
#endif
}

int
_nbd_zero_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_ZERO
  int ret;

  ret = nbd_zero (h, count, offset, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_ZERO
  missing_function (err, "zero");
  return -1;
#endif
}

int
_nbd_block_status_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        nbd_extent_callback extent_callback, uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_BLOCK_STATUS
  int ret;

  ret = nbd_block_status (h, count, offset, extent_callback, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_BLOCK_STATUS
  missing_function (err, "block_status");
  return -1;
#endif
}

int
_nbd_block_status_64_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        nbd_extent64_callback extent64_callback, uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_BLOCK_STATUS_64
  int ret;

  ret = nbd_block_status_64 (h, count, offset, extent64_callback, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_BLOCK_STATUS_64
  missing_function (err, "block_status_64");
  return -1;
#endif
}

int
_nbd_block_status_filter_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        char **contexts, nbd_extent64_callback extent64_callback,
        uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_BLOCK_STATUS_FILTER
  int ret;

  ret = nbd_block_status_filter (h, count, offset, contexts,
                                 extent64_callback, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_BLOCK_STATUS_FILTER
  missing_function (err, "block_status_filter");
  return -1;
#endif
}

int
_nbd_poll_wrapper (struct error *err,
        struct nbd_handle *h, int timeout)
{
#ifdef LIBNBD_HAVE_NBD_POLL
  int ret;

  ret = nbd_poll (h, timeout);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_POLL
  missing_function (err, "poll");
  return -1;
#endif
}

int
_nbd_poll2_wrapper (struct error *err,
        struct nbd_handle *h, int fd, int timeout)
{
#ifdef LIBNBD_HAVE_NBD_POLL2
  int ret;

  ret = nbd_poll2 (h, fd, timeout);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_POLL2
  missing_function (err, "poll2");
  return -1;
#endif
}

int
_nbd_aio_connect_wrapper (struct error *err,
        struct nbd_handle *h, const struct sockaddr *addr,
        socklen_t addrlen)
{
#ifdef LIBNBD_HAVE_NBD_AIO_CONNECT
  int ret;

  ret = nbd_aio_connect (h, addr, addrlen);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_CONNECT
  missing_function (err, "aio_connect");
  return -1;
#endif
}

int
_nbd_aio_connect_uri_wrapper (struct error *err,
        struct nbd_handle *h, const char *uri)
{
#ifdef LIBNBD_HAVE_NBD_AIO_CONNECT_URI
  int ret;

  ret = nbd_aio_connect_uri (h, uri);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_CONNECT_URI
  missing_function (err, "aio_connect_uri");
  return -1;
#endif
}

int
_nbd_aio_connect_unix_wrapper (struct error *err,
        struct nbd_handle *h, const char *unixsocket)
{
#ifdef LIBNBD_HAVE_NBD_AIO_CONNECT_UNIX
  int ret;

  ret = nbd_aio_connect_unix (h, unixsocket);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_CONNECT_UNIX
  missing_function (err, "aio_connect_unix");
  return -1;
#endif
}

int
_nbd_aio_connect_vsock_wrapper (struct error *err,
        struct nbd_handle *h, uint32_t cid, uint32_t port)
{
#ifdef LIBNBD_HAVE_NBD_AIO_CONNECT_VSOCK
  int ret;

  ret = nbd_aio_connect_vsock (h, cid, port);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_CONNECT_VSOCK
  missing_function (err, "aio_connect_vsock");
  return -1;
#endif
}

int
_nbd_aio_connect_tcp_wrapper (struct error *err,
        struct nbd_handle *h, const char *hostname, const char *port)
{
#ifdef LIBNBD_HAVE_NBD_AIO_CONNECT_TCP
  int ret;

  ret = nbd_aio_connect_tcp (h, hostname, port);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_CONNECT_TCP
  missing_function (err, "aio_connect_tcp");
  return -1;
#endif
}

int
_nbd_aio_connect_socket_wrapper (struct error *err,
        struct nbd_handle *h, int sock)
{
#ifdef LIBNBD_HAVE_NBD_AIO_CONNECT_SOCKET
  int ret;

  ret = nbd_aio_connect_socket (h, sock);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_CONNECT_SOCKET
  missing_function (err, "aio_connect_socket");
  return -1;
#endif
}

int
_nbd_aio_connect_command_wrapper (struct error *err,
        struct nbd_handle *h, char **argv)
{
#ifdef LIBNBD_HAVE_NBD_AIO_CONNECT_COMMAND
  int ret;

  ret = nbd_aio_connect_command (h, argv);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_CONNECT_COMMAND
  missing_function (err, "aio_connect_command");
  return -1;
#endif
}

int
_nbd_aio_connect_systemd_socket_activation_wrapper (struct error *err,
        struct nbd_handle *h, char **argv)
{
#ifdef LIBNBD_HAVE_NBD_AIO_CONNECT_SYSTEMD_SOCKET_ACTIVATION
  int ret;

  ret = nbd_aio_connect_systemd_socket_activation (h, argv);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_CONNECT_SYSTEMD_SOCKET_ACTIVATION
  missing_function (err, "aio_connect_systemd_socket_activation");
  return -1;
#endif
}

int
_nbd_aio_opt_go_wrapper (struct error *err,
        struct nbd_handle *h, nbd_completion_callback completion_callback)
{
#ifdef LIBNBD_HAVE_NBD_AIO_OPT_GO
  int ret;

  ret = nbd_aio_opt_go (h, completion_callback);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_OPT_GO
  missing_function (err, "aio_opt_go");
  return -1;
#endif
}

int
_nbd_aio_opt_abort_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_AIO_OPT_ABORT
  int ret;

  ret = nbd_aio_opt_abort (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_OPT_ABORT
  missing_function (err, "aio_opt_abort");
  return -1;
#endif
}

int
_nbd_aio_opt_starttls_wrapper (struct error *err,
        struct nbd_handle *h, nbd_completion_callback completion_callback)
{
#ifdef LIBNBD_HAVE_NBD_AIO_OPT_STARTTLS
  int ret;

  ret = nbd_aio_opt_starttls (h, completion_callback);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_OPT_STARTTLS
  missing_function (err, "aio_opt_starttls");
  return -1;
#endif
}

int
_nbd_aio_opt_extended_headers_wrapper (struct error *err,
        struct nbd_handle *h, nbd_completion_callback completion_callback)
{
#ifdef LIBNBD_HAVE_NBD_AIO_OPT_EXTENDED_HEADERS
  int ret;

  ret = nbd_aio_opt_extended_headers (h, completion_callback);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_OPT_EXTENDED_HEADERS
  missing_function (err, "aio_opt_extended_headers");
  return -1;
#endif
}

int
_nbd_aio_opt_structured_reply_wrapper (struct error *err,
        struct nbd_handle *h, nbd_completion_callback completion_callback)
{
#ifdef LIBNBD_HAVE_NBD_AIO_OPT_STRUCTURED_REPLY
  int ret;

  ret = nbd_aio_opt_structured_reply (h, completion_callback);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_OPT_STRUCTURED_REPLY
  missing_function (err, "aio_opt_structured_reply");
  return -1;
#endif
}

int
_nbd_aio_opt_list_wrapper (struct error *err,
        struct nbd_handle *h, nbd_list_callback list_callback,
        nbd_completion_callback completion_callback)
{
#ifdef LIBNBD_HAVE_NBD_AIO_OPT_LIST
  int ret;

  ret = nbd_aio_opt_list (h, list_callback, completion_callback);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_OPT_LIST
  missing_function (err, "aio_opt_list");
  return -1;
#endif
}

int
_nbd_aio_opt_info_wrapper (struct error *err,
        struct nbd_handle *h, nbd_completion_callback completion_callback)
{
#ifdef LIBNBD_HAVE_NBD_AIO_OPT_INFO
  int ret;

  ret = nbd_aio_opt_info (h, completion_callback);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_OPT_INFO
  missing_function (err, "aio_opt_info");
  return -1;
#endif
}

int
_nbd_aio_opt_list_meta_context_wrapper (struct error *err,
        struct nbd_handle *h, nbd_context_callback context_callback,
        nbd_completion_callback completion_callback)
{
#ifdef LIBNBD_HAVE_NBD_AIO_OPT_LIST_META_CONTEXT
  int ret;

  ret = nbd_aio_opt_list_meta_context (h, context_callback,
                                       completion_callback);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_OPT_LIST_META_CONTEXT
  missing_function (err, "aio_opt_list_meta_context");
  return -1;
#endif
}

int
_nbd_aio_opt_list_meta_context_queries_wrapper (struct error *err,
        struct nbd_handle *h, char **queries,
        nbd_context_callback context_callback,
        nbd_completion_callback completion_callback)
{
#ifdef LIBNBD_HAVE_NBD_AIO_OPT_LIST_META_CONTEXT_QUERIES
  int ret;

  ret = nbd_aio_opt_list_meta_context_queries (h, queries, context_callback,
                                               completion_callback);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_OPT_LIST_META_CONTEXT_QUERIES
  missing_function (err, "aio_opt_list_meta_context_queries");
  return -1;
#endif
}

int
_nbd_aio_opt_set_meta_context_wrapper (struct error *err,
        struct nbd_handle *h, nbd_context_callback context_callback,
        nbd_completion_callback completion_callback)
{
#ifdef LIBNBD_HAVE_NBD_AIO_OPT_SET_META_CONTEXT
  int ret;

  ret = nbd_aio_opt_set_meta_context (h, context_callback,
                                      completion_callback);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_OPT_SET_META_CONTEXT
  missing_function (err, "aio_opt_set_meta_context");
  return -1;
#endif
}

int
_nbd_aio_opt_set_meta_context_queries_wrapper (struct error *err,
        struct nbd_handle *h, char **queries,
        nbd_context_callback context_callback,
        nbd_completion_callback completion_callback)
{
#ifdef LIBNBD_HAVE_NBD_AIO_OPT_SET_META_CONTEXT_QUERIES
  int ret;

  ret = nbd_aio_opt_set_meta_context_queries (h, queries, context_callback,
                                              completion_callback);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_OPT_SET_META_CONTEXT_QUERIES
  missing_function (err, "aio_opt_set_meta_context_queries");
  return -1;
#endif
}

int64_t
_nbd_aio_pread_wrapper (struct error *err,
        struct nbd_handle *h, void *buf, size_t count, uint64_t offset,
        nbd_completion_callback completion_callback, uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_AIO_PREAD
  int64_t ret;

  ret = nbd_aio_pread (h, buf, count, offset, completion_callback, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_PREAD
  missing_function (err, "aio_pread");
  return -1;
#endif
}

int64_t
_nbd_aio_pread_structured_wrapper (struct error *err,
        struct nbd_handle *h, void *buf, size_t count, uint64_t offset,
        nbd_chunk_callback chunk_callback,
        nbd_completion_callback completion_callback, uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_AIO_PREAD_STRUCTURED
  int64_t ret;

  ret = nbd_aio_pread_structured (h, buf, count, offset, chunk_callback,
                                  completion_callback, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_PREAD_STRUCTURED
  missing_function (err, "aio_pread_structured");
  return -1;
#endif
}

int64_t
_nbd_aio_pwrite_wrapper (struct error *err,
        struct nbd_handle *h, const void *buf, size_t count,
        uint64_t offset, nbd_completion_callback completion_callback,
        uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_AIO_PWRITE
  int64_t ret;

  ret = nbd_aio_pwrite (h, buf, count, offset, completion_callback, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_PWRITE
  missing_function (err, "aio_pwrite");
  return -1;
#endif
}

int
_nbd_aio_disconnect_wrapper (struct error *err,
        struct nbd_handle *h, uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_AIO_DISCONNECT
  int ret;

  ret = nbd_aio_disconnect (h, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_DISCONNECT
  missing_function (err, "aio_disconnect");
  return -1;
#endif
}

int64_t
_nbd_aio_flush_wrapper (struct error *err,
        struct nbd_handle *h, nbd_completion_callback completion_callback,
        uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_AIO_FLUSH
  int64_t ret;

  ret = nbd_aio_flush (h, completion_callback, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_FLUSH
  missing_function (err, "aio_flush");
  return -1;
#endif
}

int64_t
_nbd_aio_trim_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        nbd_completion_callback completion_callback, uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_AIO_TRIM
  int64_t ret;

  ret = nbd_aio_trim (h, count, offset, completion_callback, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_TRIM
  missing_function (err, "aio_trim");
  return -1;
#endif
}

int64_t
_nbd_aio_cache_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        nbd_completion_callback completion_callback, uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_AIO_CACHE
  int64_t ret;

  ret = nbd_aio_cache (h, count, offset, completion_callback, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_CACHE
  missing_function (err, "aio_cache");
  return -1;
#endif
}

int64_t
_nbd_aio_zero_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        nbd_completion_callback completion_callback, uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_AIO_ZERO
  int64_t ret;

  ret = nbd_aio_zero (h, count, offset, completion_callback, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_ZERO
  missing_function (err, "aio_zero");
  return -1;
#endif
}

int64_t
_nbd_aio_block_status_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        nbd_extent_callback extent_callback,
        nbd_completion_callback completion_callback, uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_AIO_BLOCK_STATUS
  int64_t ret;

  ret = nbd_aio_block_status (h, count, offset, extent_callback,
                              completion_callback, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_BLOCK_STATUS
  missing_function (err, "aio_block_status");
  return -1;
#endif
}

int64_t
_nbd_aio_block_status_64_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        nbd_extent64_callback extent64_callback,
        nbd_completion_callback completion_callback, uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_AIO_BLOCK_STATUS_64
  int64_t ret;

  ret = nbd_aio_block_status_64 (h, count, offset, extent64_callback,
                                 completion_callback, flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_BLOCK_STATUS_64
  missing_function (err, "aio_block_status_64");
  return -1;
#endif
}

int64_t
_nbd_aio_block_status_filter_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        char **contexts, nbd_extent64_callback extent64_callback,
        nbd_completion_callback completion_callback, uint32_t flags)
{
#ifdef LIBNBD_HAVE_NBD_AIO_BLOCK_STATUS_FILTER
  int64_t ret;

  ret = nbd_aio_block_status_filter (h, count, offset, contexts,
                                     extent64_callback, completion_callback,
                                     flags);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_BLOCK_STATUS_FILTER
  missing_function (err, "aio_block_status_filter");
  return -1;
#endif
}

int
_nbd_aio_get_fd_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_AIO_GET_FD
  int ret;

  ret = nbd_aio_get_fd (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_GET_FD
  missing_function (err, "aio_get_fd");
  return -1;
#endif
}

unsigned
_nbd_aio_get_direction_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_AIO_GET_DIRECTION
  unsigned ret;

  ret = nbd_aio_get_direction (h);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_GET_DIRECTION
  missing_function (err, "aio_get_direction");
#endif
}

int
_nbd_aio_notify_read_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_AIO_NOTIFY_READ
  int ret;

  ret = nbd_aio_notify_read (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_NOTIFY_READ
  missing_function (err, "aio_notify_read");
  return -1;
#endif
}

int
_nbd_aio_notify_write_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_AIO_NOTIFY_WRITE
  int ret;

  ret = nbd_aio_notify_write (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_NOTIFY_WRITE
  missing_function (err, "aio_notify_write");
  return -1;
#endif
}

int
_nbd_aio_is_created_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_AIO_IS_CREATED
  int ret;

  ret = nbd_aio_is_created (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_IS_CREATED
  missing_function (err, "aio_is_created");
  return -1;
#endif
}

int
_nbd_aio_is_connecting_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_AIO_IS_CONNECTING
  int ret;

  ret = nbd_aio_is_connecting (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_IS_CONNECTING
  missing_function (err, "aio_is_connecting");
  return -1;
#endif
}

int
_nbd_aio_is_negotiating_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_AIO_IS_NEGOTIATING
  int ret;

  ret = nbd_aio_is_negotiating (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_IS_NEGOTIATING
  missing_function (err, "aio_is_negotiating");
  return -1;
#endif
}

int
_nbd_aio_is_ready_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_AIO_IS_READY
  int ret;

  ret = nbd_aio_is_ready (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_IS_READY
  missing_function (err, "aio_is_ready");
  return -1;
#endif
}

int
_nbd_aio_is_processing_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_AIO_IS_PROCESSING
  int ret;

  ret = nbd_aio_is_processing (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_IS_PROCESSING
  missing_function (err, "aio_is_processing");
  return -1;
#endif
}

int
_nbd_aio_is_dead_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_AIO_IS_DEAD
  int ret;

  ret = nbd_aio_is_dead (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_IS_DEAD
  missing_function (err, "aio_is_dead");
  return -1;
#endif
}

int
_nbd_aio_is_closed_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_AIO_IS_CLOSED
  int ret;

  ret = nbd_aio_is_closed (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_IS_CLOSED
  missing_function (err, "aio_is_closed");
  return -1;
#endif
}

int
_nbd_aio_command_completed_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t cookie)
{
#ifdef LIBNBD_HAVE_NBD_AIO_COMMAND_COMPLETED
  int ret;

  ret = nbd_aio_command_completed (h, cookie);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_COMMAND_COMPLETED
  missing_function (err, "aio_command_completed");
  return -1;
#endif
}

int64_t
_nbd_aio_peek_command_completed_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_AIO_PEEK_COMMAND_COMPLETED
  int64_t ret;

  ret = nbd_aio_peek_command_completed (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_PEEK_COMMAND_COMPLETED
  missing_function (err, "aio_peek_command_completed");
  return -1;
#endif
}

int
_nbd_aio_in_flight_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_AIO_IN_FLIGHT
  int ret;

  ret = nbd_aio_in_flight (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_AIO_IN_FLIGHT
  missing_function (err, "aio_in_flight");
  return -1;
#endif
}

const char *
_nbd_connection_state_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_CONNECTION_STATE
  const char * ret;

  ret = nbd_connection_state (h);
  if (ret == NULL)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_CONNECTION_STATE
  missing_function (err, "connection_state");
  return NULL;
#endif
}

const char *
_nbd_get_package_name_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_PACKAGE_NAME
  const char * ret;

  ret = nbd_get_package_name (h);
  if (ret == NULL)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_PACKAGE_NAME
  missing_function (err, "get_package_name");
  return NULL;
#endif
}

const char *
_nbd_get_version_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_VERSION
  const char * ret;

  ret = nbd_get_version (h);
  if (ret == NULL)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_VERSION
  missing_function (err, "get_version");
  return NULL;
#endif
}

int
_nbd_kill_subprocess_wrapper (struct error *err,
        struct nbd_handle *h, int signum)
{
#ifdef LIBNBD_HAVE_NBD_KILL_SUBPROCESS
  int ret;

  ret = nbd_kill_subprocess (h, signum);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_KILL_SUBPROCESS
  missing_function (err, "kill_subprocess");
  return -1;
#endif
}

int
_nbd_supports_tls_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_SUPPORTS_TLS
  int ret;

  ret = nbd_supports_tls (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SUPPORTS_TLS
  missing_function (err, "supports_tls");
  return -1;
#endif
}

int
_nbd_supports_vsock_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_SUPPORTS_VSOCK
  int ret;

  ret = nbd_supports_vsock (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SUPPORTS_VSOCK
  missing_function (err, "supports_vsock");
  return -1;
#endif
}

int
_nbd_supports_uri_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_SUPPORTS_URI
  int ret;

  ret = nbd_supports_uri (h);
  if (ret == -1)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_SUPPORTS_URI
  missing_function (err, "supports_uri");
  return -1;
#endif
}

char *
_nbd_get_uri_wrapper (struct error *err,
        struct nbd_handle *h)
{
#ifdef LIBNBD_HAVE_NBD_GET_URI
  char * ret;

  ret = nbd_get_uri (h);
  if (ret == NULL)
    save_error (err);
  return ret;
#else // !LIBNBD_HAVE_NBD_GET_URI
  missing_function (err, "get_uri");
  return NULL;
#endif
}

int
_nbd_chunk_callback_wrapper (void *user_data, const void *subbuf,
                             size_t count, uint64_t offset, unsigned status,
                             int *error)
{
  return chunk_callback ((long)user_data, subbuf, count, offset, status, error);
}

void
_nbd_chunk_callback_free (void *user_data)
{
  long *p = user_data;
  extern void freeCallbackId (long);
  freeCallbackId (*p);
  free (p);
}

int
_nbd_completion_callback_wrapper (void *user_data, int *error)
{
  return completion_callback ((long)user_data, error);
}

void
_nbd_completion_callback_free (void *user_data)
{
  long *p = user_data;
  extern void freeCallbackId (long);
  freeCallbackId (*p);
  free (p);
}

int
_nbd_debug_callback_wrapper (void *user_data, const char *context,
                             const char *msg)
{
  return debug_callback ((long)user_data, context, msg);
}

void
_nbd_debug_callback_free (void *user_data)
{
  long *p = user_data;
  extern void freeCallbackId (long);
  freeCallbackId (*p);
  free (p);
}

int
_nbd_extent_callback_wrapper (void *user_data, const char *metacontext,
                              uint64_t offset, uint32_t *entries,
                              size_t nr_entries, int *error)
{
  return extent_callback ((long)user_data, metacontext, offset, entries, nr_entries, error);
}

void
_nbd_extent_callback_free (void *user_data)
{
  long *p = user_data;
  extern void freeCallbackId (long);
  freeCallbackId (*p);
  free (p);
}

int
_nbd_extent64_callback_wrapper (void *user_data, const char *metacontext,
                                uint64_t offset, nbd_extent *entries,
                                size_t nr_entries, int *error)
{
  return extent64_callback ((long)user_data, metacontext, offset, entries, nr_entries, error);
}

void
_nbd_extent64_callback_free (void *user_data)
{
  long *p = user_data;
  extern void freeCallbackId (long);
  freeCallbackId (*p);
  free (p);
}

int
_nbd_list_callback_wrapper (void *user_data, const char *name,
                            const char *description)
{
  return list_callback ((long)user_data, name, description);
}

void
_nbd_list_callback_free (void *user_data)
{
  long *p = user_data;
  extern void freeCallbackId (long);
  freeCallbackId (*p);
  free (p);
}

int
_nbd_context_callback_wrapper (void *user_data, const char *name)
{
  return context_callback ((long)user_data, name);
}

void
_nbd_context_callback_free (void *user_data)
{
  long *p = user_data;
  extern void freeCallbackId (long);
  freeCallbackId (*p);
  free (p);
}

// There must be no blank line between end comment and import!
// https://github.com/golang/go/issues/9733
*/
import "C"
