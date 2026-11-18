// Optional drop-in mapping: array_* -> zarray_*.
//
// Usage: compile with -DZARRAY_COMPAT_NAMES and include this header after
// <collectc/array.h>. Do NOT enable it in a TU that also needs to call the
// cltc implementation under the plain names.

#ifndef ZARRAY_COMPAT_H
#define ZARRAY_COMPAT_H

#ifdef ZARRAY_COMPAT_NAMES
#define array_new              zarray_new
#define array_new_conf         zarray_new_conf
#define array_conf_init        zarray_conf_init
#define array_destroy          zarray_destroy
#define array_destroy_cb       zarray_destroy_cb
#define array_add              zarray_add
#define array_add_at           zarray_add_at
#define array_replace_at       zarray_replace_at
#define array_swap_at          zarray_swap_at
#define array_remove           zarray_remove
#define array_remove_at        zarray_remove_at
#define array_remove_last      zarray_remove_last
#define array_remove_all       zarray_remove_all
#define array_remove_all_free  zarray_remove_all_free
#define array_get_at           zarray_get_at
#define array_get_last         zarray_get_last
#define array_subarray         zarray_subarray
#define array_copy_shallow     zarray_copy_shallow
#define array_copy_deep        zarray_copy_deep
#define array_reverse_         zarray_reverse_
#define array_trim_capacity    zarray_trim_capacity
#define array_contains         zarray_contains
#define array_contains_value   zarray_contains_value
#define array_size             zarray_size
#define array_capacity         zarray_capacity
#define array_index_of         zarray_index_of
#define array_sort             zarray_sort
#define array_map              zarray_map
#define array_reduce           zarray_reduce
#define array_filter_mut       zarray_filter_mut
#define array_filter           zarray_filter
#define array_iter_init        zarray_iter_init
#define array_iter_next        zarray_iter_next
#define array_iter_remove      zarray_iter_remove
#define array_iter_add         zarray_iter_add
#define array_iter_replace     zarray_iter_replace
#define array_iter_index       zarray_iter_index
#define array_zip_iter_init    zarray_zip_iter_init
#define array_zip_iter_next    zarray_zip_iter_next
#define array_zip_iter_add     zarray_zip_iter_add
#define array_zip_iter_remove  zarray_zip_iter_remove
#define array_zip_iter_replace zarray_zip_iter_replace
#define array_zip_iter_index   zarray_zip_iter_index
#define array_get_buffer       zarray_get_buffer
#endif /* ZARRAY_COMPAT_NAMES */

#endif /* ZARRAY_COMPAT_H */
