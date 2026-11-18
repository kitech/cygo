// zarray.h — Zig implementation of collectc array.h (same params/returns).
//
// Implemented in zigsrc/corona_zig.zig (module `zarray`), linked into libcrn.
// Symbol prefix is `zarray_*` so it can coexist with cltc's array.c; define
// ZARRAY_COMPAT_NAMES and include zarray_compat.h to map array_* -> zarray_*.

#ifndef ZARRAY_H
#define ZARRAY_H

#include <stdbool.h>
#include <stddef.h>
#include "collectc/array.h" /* cc_stat, ArrayConf, ArrayIter, ArrayZipIter */

typedef struct zarray_s ZArray;

enum cc_stat zarray_new             (ZArray **out);
enum cc_stat zarray_new_conf        (ArrayConf const * const conf, ZArray **out);
void         zarray_conf_init       (ArrayConf *conf);

void         zarray_destroy         (ZArray *ar);
void         zarray_destroy_cb      (ZArray *ar, void (*cb) (void*));

enum cc_stat zarray_add             (ZArray *ar, void *element);
enum cc_stat zarray_add_at          (ZArray *ar, void *element, size_t index);
enum cc_stat zarray_replace_at      (ZArray *ar, void *element, size_t index, void **out);
enum cc_stat zarray_swap_at         (ZArray *ar, size_t index1, size_t index2);

enum cc_stat zarray_remove          (ZArray *ar, void *element, void **out);
enum cc_stat zarray_remove_at       (ZArray *ar, size_t index, void **out);
enum cc_stat zarray_remove_last     (ZArray *ar, void **out);
void         zarray_remove_all      (ZArray *ar);
void         zarray_remove_all_free (ZArray *ar);

enum cc_stat zarray_get_at          (ZArray *ar, size_t index, void **out);
enum cc_stat zarray_get_last        (ZArray *ar, void **out);

enum cc_stat zarray_subarray        (ZArray *ar, size_t from, size_t to, ZArray **out);
enum cc_stat zarray_copy_shallow    (ZArray *ar, ZArray **out);
enum cc_stat zarray_copy_deep       (ZArray *ar, void *(*cp) (void*), ZArray **out);

void         zarray_reverse_        (ZArray *ar);
enum cc_stat zarray_trim_capacity   (ZArray *ar);

size_t       zarray_contains        (ZArray *ar, void *element);
size_t       zarray_contains_value  (ZArray *ar, void *element, int (*cmp) (const void*, const void*));
size_t       zarray_size            (ZArray *ar);
size_t       zarray_capacity        (ZArray *ar);

enum cc_stat zarray_index_of        (ZArray *ar, void *element, size_t *index);
void         zarray_sort            (ZArray *ar, int (*cmp) (const void*, const void*));

void         zarray_map             (ZArray *ar, void (*fn) (void*));
void         zarray_reduce          (ZArray *ar, void (*fn) (void*, void*, void*), void *result);

enum cc_stat zarray_filter_mut      (ZArray *ar, bool (*predicate) (const void*));
enum cc_stat zarray_filter          (ZArray *ar, bool (*predicate) (const void*), ZArray **out);

void         zarray_iter_init       (ArrayIter *iter, ZArray *ar);
enum cc_stat zarray_iter_next       (ArrayIter *iter, void **out);
enum cc_stat zarray_iter_remove     (ArrayIter *iter, void **out);
enum cc_stat zarray_iter_add        (ArrayIter *iter, void *element);
enum cc_stat zarray_iter_replace    (ArrayIter *iter, void *element, void **out);
size_t       zarray_iter_index      (ArrayIter *iter);

void         zarray_zip_iter_init   (ArrayZipIter *iter, ZArray *a1, ZArray *a2);
enum cc_stat zarray_zip_iter_next   (ArrayZipIter *iter, void **out1, void **out2);
enum cc_stat zarray_zip_iter_add    (ArrayZipIter *iter, void *e1, void *e2);
enum cc_stat zarray_zip_iter_remove (ArrayZipIter *iter, void **out1, void **out2);
enum cc_stat zarray_zip_iter_replace(ArrayZipIter *iter, void *e1, void *e2, void **out1, void **out2);
size_t       zarray_zip_iter_index  (ArrayZipIter *iter);

const void* const* zarray_get_buffer(ZArray *ar);

#endif /* ZARRAY_H */
