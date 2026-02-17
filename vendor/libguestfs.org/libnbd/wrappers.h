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

#ifndef LIBNBD_GOLANG_WRAPPERS_H
#define LIBNBD_GOLANG_WRAPPERS_H

#include <stdio.h>
#include <string.h>
#include <assert.h>
#include <errno.h>

#include "libnbd.h"

/* When calling callbacks we pass the callback ID (a golang int /
 * C.long) in the void *user_data field.  We need to create a block
 * to store the callback number.  This must be freed by C.free(vp)
 */
static inline void *
alloc_cbid (long i)
{
  long *p = malloc (sizeof (long));
  assert (p != NULL);
  *p = i;
  return p;
}

/* save_error is called from the same thread to make a copy
 * of the error which can later be retrieve from golang code
 * possibly running in a different thread.
 */
struct error {
  char *error;
  int errnum;
};

static inline void
save_error (struct error *err)
{
  err->error = strdup (nbd_get_error ());
  err->errnum = nbd_get_errno ();
}

static inline void
free_error (struct error *err)
{
  free (err->error);
}

/* If you mix old C library and new bindings then some C
 * functions may not be defined.  They return ENOTSUP.
 */
static inline void
missing_function (struct error *err, const char *fn)
{
  asprintf (&err->error, "%s: "
            "function missing because golang bindings were compiled "
            "against an old version of the C library", fn);
  err->errnum = ENOTSUP;
}

int _nbd_set_debug_wrapper (struct error *err,
        struct nbd_handle *h, bool debug);
int _nbd_get_debug_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_set_debug_callback_wrapper (struct error *err,
        struct nbd_handle *h, nbd_debug_callback debug_callback);
int _nbd_clear_debug_callback_wrapper (struct error *err,
        struct nbd_handle *h);
uint64_t _nbd_stats_bytes_sent_wrapper (struct error *err,
        struct nbd_handle *h);
uint64_t _nbd_stats_chunks_sent_wrapper (struct error *err,
        struct nbd_handle *h);
uint64_t _nbd_stats_bytes_received_wrapper (struct error *err,
        struct nbd_handle *h);
uint64_t _nbd_stats_chunks_received_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_set_handle_name_wrapper (struct error *err,
        struct nbd_handle *h, const char *handle_name);
char * _nbd_get_handle_name_wrapper (struct error *err,
        struct nbd_handle *h);
uintptr_t _nbd_set_private_data_wrapper (struct error *err,
        struct nbd_handle *h, uintptr_t private_data);
uintptr_t _nbd_get_private_data_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_set_export_name_wrapper (struct error *err,
        struct nbd_handle *h, const char *export_name);
char * _nbd_get_export_name_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_set_request_block_size_wrapper (struct error *err,
        struct nbd_handle *h, bool request);
int _nbd_get_request_block_size_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_set_full_info_wrapper (struct error *err,
        struct nbd_handle *h, bool request);
int _nbd_get_full_info_wrapper (struct error *err,
        struct nbd_handle *h);
char * _nbd_get_canonical_export_name_wrapper (struct error *err,
        struct nbd_handle *h);
char * _nbd_get_export_description_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_set_tls_wrapper (struct error *err,
        struct nbd_handle *h, int tls);
int _nbd_get_tls_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_get_tls_negotiated_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_set_tls_certificates_wrapper (struct error *err,
        struct nbd_handle *h, const char *dir);
int _nbd_set_tls_verify_peer_wrapper (struct error *err,
        struct nbd_handle *h, bool verify);
int _nbd_get_tls_verify_peer_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_set_tls_username_wrapper (struct error *err,
        struct nbd_handle *h, const char *username);
char * _nbd_get_tls_username_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_set_tls_psk_file_wrapper (struct error *err,
        struct nbd_handle *h, const char *filename);
int _nbd_set_request_extended_headers_wrapper (struct error *err,
        struct nbd_handle *h, bool request);
int _nbd_get_request_extended_headers_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_get_extended_headers_negotiated_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_set_request_structured_replies_wrapper (struct error *err,
        struct nbd_handle *h, bool request);
int _nbd_get_request_structured_replies_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_get_structured_replies_negotiated_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_set_request_meta_context_wrapper (struct error *err,
        struct nbd_handle *h, bool request);
int _nbd_get_request_meta_context_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_set_handshake_flags_wrapper (struct error *err,
        struct nbd_handle *h, uint32_t flags);
uint32_t _nbd_get_handshake_flags_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_set_pread_initialize_wrapper (struct error *err,
        struct nbd_handle *h, bool request);
int _nbd_get_pread_initialize_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_set_strict_mode_wrapper (struct error *err,
        struct nbd_handle *h, uint32_t flags);
uint32_t _nbd_get_strict_mode_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_set_opt_mode_wrapper (struct error *err,
        struct nbd_handle *h, bool enable);
int _nbd_get_opt_mode_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_opt_go_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_opt_abort_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_opt_starttls_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_opt_extended_headers_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_opt_structured_reply_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_opt_list_wrapper (struct error *err,
        struct nbd_handle *h, nbd_list_callback list_callback);
int _nbd_opt_info_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_opt_list_meta_context_wrapper (struct error *err,
        struct nbd_handle *h, nbd_context_callback context_callback);
int _nbd_opt_list_meta_context_queries_wrapper (struct error *err,
        struct nbd_handle *h, char **queries,
        nbd_context_callback context_callback);
int _nbd_opt_set_meta_context_wrapper (struct error *err,
        struct nbd_handle *h, nbd_context_callback context_callback);
int _nbd_opt_set_meta_context_queries_wrapper (struct error *err,
        struct nbd_handle *h, char **queries,
        nbd_context_callback context_callback);
int _nbd_add_meta_context_wrapper (struct error *err,
        struct nbd_handle *h, const char *name);
ssize_t _nbd_get_nr_meta_contexts_wrapper (struct error *err,
        struct nbd_handle *h);
char * _nbd_get_meta_context_wrapper (struct error *err,
        struct nbd_handle *h, size_t i);
int _nbd_clear_meta_contexts_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_set_uri_allow_transports_wrapper (struct error *err,
        struct nbd_handle *h, uint32_t mask);
int _nbd_set_uri_allow_tls_wrapper (struct error *err,
        struct nbd_handle *h, int tls);
int _nbd_set_uri_allow_local_file_wrapper (struct error *err,
        struct nbd_handle *h, bool allow);
int _nbd_connect_uri_wrapper (struct error *err,
        struct nbd_handle *h, const char *uri);
int _nbd_connect_unix_wrapper (struct error *err,
        struct nbd_handle *h, const char *unixsocket);
int _nbd_connect_vsock_wrapper (struct error *err,
        struct nbd_handle *h, uint32_t cid, uint32_t port);
int _nbd_connect_tcp_wrapper (struct error *err,
        struct nbd_handle *h, const char *hostname, const char *port);
int _nbd_connect_socket_wrapper (struct error *err,
        struct nbd_handle *h, int sock);
int _nbd_connect_command_wrapper (struct error *err,
        struct nbd_handle *h, char **argv);
int _nbd_connect_systemd_socket_activation_wrapper (struct error *err,
        struct nbd_handle *h, char **argv);
int _nbd_set_socket_activation_name_wrapper (struct error *err,
        struct nbd_handle *h, const char *socket_name);
char * _nbd_get_socket_activation_name_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_is_read_only_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_can_flush_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_can_fua_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_is_rotational_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_can_trim_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_can_zero_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_can_fast_zero_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_can_block_status_payload_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_can_df_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_can_multi_conn_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_can_cache_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_can_meta_context_wrapper (struct error *err,
        struct nbd_handle *h, const char *metacontext);
const char * _nbd_get_protocol_wrapper (struct error *err,
        struct nbd_handle *h);
int64_t _nbd_get_size_wrapper (struct error *err,
        struct nbd_handle *h);
int64_t _nbd_get_block_size_wrapper (struct error *err,
        struct nbd_handle *h, int size_type);
int _nbd_pread_wrapper (struct error *err,
        struct nbd_handle *h, void *buf, size_t count, uint64_t offset,
        uint32_t flags);
int _nbd_pread_structured_wrapper (struct error *err,
        struct nbd_handle *h, void *buf, size_t count, uint64_t offset,
        nbd_chunk_callback chunk_callback, uint32_t flags);
int _nbd_pwrite_wrapper (struct error *err,
        struct nbd_handle *h, const void *buf, size_t count,
        uint64_t offset, uint32_t flags);
int _nbd_shutdown_wrapper (struct error *err,
        struct nbd_handle *h, uint32_t flags);
int _nbd_flush_wrapper (struct error *err,
        struct nbd_handle *h, uint32_t flags);
int _nbd_trim_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        uint32_t flags);
int _nbd_cache_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        uint32_t flags);
int _nbd_zero_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        uint32_t flags);
int _nbd_block_status_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        nbd_extent_callback extent_callback, uint32_t flags);
int _nbd_block_status_64_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        nbd_extent64_callback extent64_callback, uint32_t flags);
int _nbd_block_status_filter_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        char **contexts, nbd_extent64_callback extent64_callback,
        uint32_t flags);
int _nbd_poll_wrapper (struct error *err,
        struct nbd_handle *h, int timeout);
int _nbd_poll2_wrapper (struct error *err,
        struct nbd_handle *h, int fd, int timeout);
int _nbd_aio_connect_wrapper (struct error *err,
        struct nbd_handle *h, const struct sockaddr *addr,
        socklen_t addrlen);
int _nbd_aio_connect_uri_wrapper (struct error *err,
        struct nbd_handle *h, const char *uri);
int _nbd_aio_connect_unix_wrapper (struct error *err,
        struct nbd_handle *h, const char *unixsocket);
int _nbd_aio_connect_vsock_wrapper (struct error *err,
        struct nbd_handle *h, uint32_t cid, uint32_t port);
int _nbd_aio_connect_tcp_wrapper (struct error *err,
        struct nbd_handle *h, const char *hostname, const char *port);
int _nbd_aio_connect_socket_wrapper (struct error *err,
        struct nbd_handle *h, int sock);
int _nbd_aio_connect_command_wrapper (struct error *err,
        struct nbd_handle *h, char **argv);
int _nbd_aio_connect_systemd_socket_activation_wrapper (struct error *err,
        struct nbd_handle *h, char **argv);
int _nbd_aio_opt_go_wrapper (struct error *err,
        struct nbd_handle *h, nbd_completion_callback completion_callback);
int _nbd_aio_opt_abort_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_aio_opt_starttls_wrapper (struct error *err,
        struct nbd_handle *h, nbd_completion_callback completion_callback);
int _nbd_aio_opt_extended_headers_wrapper (struct error *err,
        struct nbd_handle *h, nbd_completion_callback completion_callback);
int _nbd_aio_opt_structured_reply_wrapper (struct error *err,
        struct nbd_handle *h, nbd_completion_callback completion_callback);
int _nbd_aio_opt_list_wrapper (struct error *err,
        struct nbd_handle *h, nbd_list_callback list_callback,
        nbd_completion_callback completion_callback);
int _nbd_aio_opt_info_wrapper (struct error *err,
        struct nbd_handle *h, nbd_completion_callback completion_callback);
int _nbd_aio_opt_list_meta_context_wrapper (struct error *err,
        struct nbd_handle *h, nbd_context_callback context_callback,
        nbd_completion_callback completion_callback);
int _nbd_aio_opt_list_meta_context_queries_wrapper (struct error *err,
        struct nbd_handle *h, char **queries,
        nbd_context_callback context_callback,
        nbd_completion_callback completion_callback);
int _nbd_aio_opt_set_meta_context_wrapper (struct error *err,
        struct nbd_handle *h, nbd_context_callback context_callback,
        nbd_completion_callback completion_callback);
int _nbd_aio_opt_set_meta_context_queries_wrapper (struct error *err,
        struct nbd_handle *h, char **queries,
        nbd_context_callback context_callback,
        nbd_completion_callback completion_callback);
int64_t _nbd_aio_pread_wrapper (struct error *err,
        struct nbd_handle *h, void *buf, size_t count, uint64_t offset,
        nbd_completion_callback completion_callback, uint32_t flags);
int64_t _nbd_aio_pread_structured_wrapper (struct error *err,
        struct nbd_handle *h, void *buf, size_t count, uint64_t offset,
        nbd_chunk_callback chunk_callback,
        nbd_completion_callback completion_callback, uint32_t flags);
int64_t _nbd_aio_pwrite_wrapper (struct error *err,
        struct nbd_handle *h, const void *buf, size_t count,
        uint64_t offset, nbd_completion_callback completion_callback,
        uint32_t flags);
int _nbd_aio_disconnect_wrapper (struct error *err,
        struct nbd_handle *h, uint32_t flags);
int64_t _nbd_aio_flush_wrapper (struct error *err,
        struct nbd_handle *h, nbd_completion_callback completion_callback,
        uint32_t flags);
int64_t _nbd_aio_trim_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        nbd_completion_callback completion_callback, uint32_t flags);
int64_t _nbd_aio_cache_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        nbd_completion_callback completion_callback, uint32_t flags);
int64_t _nbd_aio_zero_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        nbd_completion_callback completion_callback, uint32_t flags);
int64_t _nbd_aio_block_status_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        nbd_extent_callback extent_callback,
        nbd_completion_callback completion_callback, uint32_t flags);
int64_t _nbd_aio_block_status_64_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        nbd_extent64_callback extent64_callback,
        nbd_completion_callback completion_callback, uint32_t flags);
int64_t _nbd_aio_block_status_filter_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t count, uint64_t offset,
        char **contexts, nbd_extent64_callback extent64_callback,
        nbd_completion_callback completion_callback, uint32_t flags);
int _nbd_aio_get_fd_wrapper (struct error *err,
        struct nbd_handle *h);
unsigned _nbd_aio_get_direction_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_aio_notify_read_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_aio_notify_write_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_aio_is_created_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_aio_is_connecting_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_aio_is_negotiating_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_aio_is_ready_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_aio_is_processing_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_aio_is_dead_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_aio_is_closed_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_aio_command_completed_wrapper (struct error *err,
        struct nbd_handle *h, uint64_t cookie);
int64_t _nbd_aio_peek_command_completed_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_aio_in_flight_wrapper (struct error *err,
        struct nbd_handle *h);
const char * _nbd_connection_state_wrapper (struct error *err,
        struct nbd_handle *h);
const char * _nbd_get_package_name_wrapper (struct error *err,
        struct nbd_handle *h);
const char * _nbd_get_version_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_kill_subprocess_wrapper (struct error *err,
        struct nbd_handle *h, int signum);
int _nbd_supports_tls_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_supports_vsock_wrapper (struct error *err,
        struct nbd_handle *h);
int _nbd_supports_uri_wrapper (struct error *err,
        struct nbd_handle *h);
char * _nbd_get_uri_wrapper (struct error *err,
        struct nbd_handle *h);

extern int chunk_callback ();

int _nbd_chunk_callback_wrapper (void *user_data, const void *subbuf,
                                 size_t count, uint64_t offset,
                                 unsigned status, int *error);
void _nbd_chunk_callback_free (void *user_data);

extern int completion_callback ();

int _nbd_completion_callback_wrapper (void *user_data, int *error);
void _nbd_completion_callback_free (void *user_data);

extern int debug_callback ();

int _nbd_debug_callback_wrapper (void *user_data, const char *context,
                                 const char *msg);
void _nbd_debug_callback_free (void *user_data);

extern int extent_callback ();

int _nbd_extent_callback_wrapper (void *user_data, const char *metacontext,
                                  uint64_t offset, uint32_t *entries,
                                  size_t nr_entries, int *error);
void _nbd_extent_callback_free (void *user_data);

extern int extent64_callback ();

int _nbd_extent64_callback_wrapper (void *user_data,
                                    const char *metacontext,
                                    uint64_t offset, nbd_extent *entries,
                                    size_t nr_entries, int *error);
void _nbd_extent64_callback_free (void *user_data);

extern int list_callback ();

int _nbd_list_callback_wrapper (void *user_data, const char *name,
                                const char *description);
void _nbd_list_callback_free (void *user_data);

extern int context_callback ();

int _nbd_context_callback_wrapper (void *user_data, const char *name);
void _nbd_context_callback_free (void *user_data);

#endif /* LIBNBD_GOLANG_WRAPPERS_H */
