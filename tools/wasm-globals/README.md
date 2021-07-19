# wasm globals
List all exported symbols in a wasm module

```
 ./bin/wasm-globals tools/modules/
 ```
 

```
import env/proxy_remove_header_map_value => func
import env/proxy_replace_header_map_value => func
import env/proxy_define_metric => func
import env/proxy_dequeue_shared_queue => func
import env/proxy_resolve_shared_queue => func
import env/proxy_http_call => func
import env/proxy_log => func
import env/proxy_get_buffer_bytes => func
import env/proxy_increment_metric => func
import env/proxy_register_shared_queue => func
import wasi_unstable/fd_write => func
import env/proxy_continue_stream => func
import env/proxy_set_buffer_bytes => func
import env/proxy_get_property => func
import env/proxy_set_shared_data => func
import env/proxy_enqueue_shared_queue => func
import env/proxy_set_tick_period_milliseconds => func
import env/proxy_send_local_response => func
import env/proxy_get_metric => func
import env/proxy_get_header_map_pairs => func
import env/proxy_get_header_map_value => func
import env/proxy_get_shared_data => func
import env/proxy_set_effective_context => func
import env/proxy_call_foreign_function => func
import wasi_unstable/clock_time_get => func


export _start => func
export memory => memory
export proxy_abi_version_0_2_0 => func
export proxy_on_configure => func
export proxy_on_context_create => func
export proxy_on_delete => func
export proxy_on_done => func
export proxy_on_downstream_connection_close => func
export proxy_on_downstream_data => func
export proxy_on_http_call_response => func
export proxy_on_log => func
export proxy_on_memory_allocate => func
export proxy_on_new_connection => func
export proxy_on_queue_ready => func
export proxy_on_request_body => func
export proxy_on_request_headers => func
export proxy_on_request_trailers => func
export proxy_on_response_body => func
export proxy_on_response_headers => func
export proxy_on_response_trailers => func
export proxy_on_tick => func
export proxy_on_upstream_connection_close => func
export proxy_on_upstream_data => func
export proxy_on_vm_start => func
```