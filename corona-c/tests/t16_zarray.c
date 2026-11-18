// t16: differential test — zarray (Zig implementation) vs cltc array (C).
// The same operation sequence runs on both; results and return codes are
// compared at every step.
#include "t00_common.h"
#include "zarray.h"
#include <collectc/array.h>

static crn_hchan* done;

// qsort semantics: a/b point at slots that hold void*
static int icmp(const void* a, const void* b) {
    int x = *(const int*)(*(void* const*)a);
    int y = *(const int*)(*(void* const*)b);
    return (x > y) - (x < y);
}
// value semantics (cltc array_contains_value passes elements on both sides)
static int vcmp(const void* a, const void* b) {
    int x = *(const int*)a;
    int y = *(const int*)b;
    return (x > y) - (x < y);
}
static bool is_even(const void* p) { return (((int)(uintptr_t)p) % 2) == 0; }
static void bump(void* p) { *(int*)p += 1; }
static void* dbl(void* p) {
    int* v = crn_gc_malloc(sizeof(int));
    *v = *(int*)p * 2;
    return v;
}
static long sum_acc;
static void fold(void* a, void* b, void* result) {
    // mirrors cltc reduce calling convention
    long* acc = result;
    if (a) *acc += *(int*)a;
    if (b) *acc += *(int*)b;
    (void)sum_acc;
}

static void compare_all(ZArray* z, Array* c, const char* where) {
    CHECK_EQ(zarray_size(z), array_size(c), "%s size", where);
    for (size_t i = 0; i < array_size(c); i++) {
        void* zv = nilptr;
        void* cv = nilptr;
        CHECK_EQ(zarray_get_at(z, i, &zv), array_get_at(c, i, &cv), "%s get rc %zu", where, i);
        CHECK(zv == cv, "%s elem %zu z=%p c=%p", where, i, zv, cv);
    }
}

static void zuser(void* arg) {
    (void)arg;

    // ---- construction ----
    ZArray* z = nilptr;
    Array* c = nilptr;
    CHECK_EQ(zarray_new(&z), CC_OK, "zarray_new");
    CHECK_EQ(array_new(&c), CC_OK, "array_new");
    compare_all(z, c, "new");

    ArrayConf conf;
    zarray_conf_init(&conf);
    CHECK_EQ(conf.capacity, 8, "conf cap=%zu", conf.capacity);
    CHECK_EQ(conf.exp_factor, 2.0f, "conf exp");

    // ---- add / add_at ----
    for (int i = 0; i < 20; i++) {
        int* v = crn_gc_malloc(sizeof(int));
        *v = i;
        CHECK_EQ(zarray_add(z, v), array_add(c, v), "add rc %d", i);
    }
    compare_all(z, c, "add");

    int* mid = crn_gc_malloc(sizeof(int));
    *mid = 999;
    CHECK_EQ(zarray_add_at(z, mid, 5), array_add_at(c, mid, 5), "add_at rc");
    compare_all(z, c, "add_at");
    CHECK_EQ(zarray_add_at(z, mid, 9999), array_add_at(c, mid, 9999), "add_at oob");

    // ---- get / last / index / contains ----
    void *zl = nilptr, *cl = nilptr;
    CHECK_EQ(zarray_get_last(z, &zl), array_get_last(c, &cl), "get_last rc");
    CHECK(zl == cl, "get_last val");
    void *zg = nilptr, *cg = nilptr;
    CHECK_EQ(zarray_get_at(z, 9999, &zg), array_get_at(c, 9999, &cg), "get oob");
    size_t zi = 123, ci = 123;
    CHECK_EQ(zarray_index_of(z, mid, &zi), array_index_of(c, mid, &ci), "index_of rc");
    CHECK_EQ(zi, ci, "index_of val");
    CHECK_EQ(zarray_contains(z, mid), array_contains(c, mid), "contains");
    CHECK_EQ(zarray_contains_value(z, mid, vcmp), array_contains_value(c, mid, vcmp), "contains_value");
    CHECK_EQ(zarray_capacity(z), array_capacity(c), "capacity after add");

    // ---- replace / swap / reverse ----
    int* nv = crn_gc_malloc(sizeof(int));
    *nv = 777;
    void *zr = nilptr, *cr = nilptr;
    CHECK_EQ(zarray_replace_at(z, nv, 3, &zr), array_replace_at(c, nv, 3, &cr), "replace rc");
    CHECK(zr == cr, "replace out");
    CHECK_EQ(zarray_swap_at(z, 0, 1), array_swap_at(c, 0, 1), "swap rc");
    zarray_reverse_(z);
    array_reverse_(c);
    compare_all(z, c, "reverse");

    // ---- remove family ----
    void *z0 = nilptr, *c0 = nilptr;
    CHECK_EQ(zarray_remove_last(z, &z0), array_remove_last(c, &c0), "remove_last rc");
    CHECK(z0 == c0, "remove_last val");
    CHECK_EQ(zarray_remove(z, nv, &z0), array_remove(c, nv, &c0), "remove rc");
    CHECK(z0 == c0, "remove val");
    CHECK_EQ(zarray_remove(z, nv, &z0), array_remove(c, nv, &c0), "remove missing");
    CHECK_EQ(zarray_remove_at(z, 0, &z0), array_remove_at(c, 0, &c0), "remove_at rc");
    CHECK(z0 == c0, "remove_at val");
    CHECK_EQ(zarray_remove_at(z, 9999, &z0), array_remove_at(c, 9999, &c0), "remove_at oob");
    compare_all(z, c, "remove");

    // ---- sort ----
    zarray_sort(z, icmp);
    array_sort(c, icmp);
    compare_all(z, c, "sort");
    for (size_t i = 1; i < array_size(c); i++) {
        void *a = nilptr, *b = nilptr;
        zarray_get_at(z, i - 1, &a);
        zarray_get_at(z, i, &b);
        CHECK(*(int*)a <= *(int*)b, "sort order %zu", i);
    }

    // ---- copy / subarray ----
    ZArray* zs = nilptr;
    Array* cs = nilptr;
    CHECK_EQ(zarray_subarray(z, 1, 3, &zs), array_subarray(c, 1, 3, &cs), "subarray rc");
    compare_all(zs, cs, "subarray");
    CHECK_EQ(zarray_subarray(z, 3, 1, &zs), array_subarray(c, 3, 1, &cs), "subarray bad range");
    CHECK_EQ(zarray_copy_shallow(z, &zs), array_copy_shallow(c, &cs), "copy_shallow rc");
    compare_all(zs, cs, "copy_shallow");
    CHECK_EQ(zarray_copy_deep(z, dbl, &zs), array_copy_deep(c, dbl, &cs), "copy_deep rc");
    // dbl allocates fresh ints per array: compare values, not pointers
    CHECK_EQ(zarray_size(zs), array_size(cs), "copy_deep size");
    for (size_t i = 0; i < array_size(cs); i++) {
        void *zv = nilptr, *cv = nilptr;
        CHECK_EQ(zarray_get_at(zs, i, &zv), array_get_at(cs, i, &cv), "copy_deep get");
        CHECK_EQ(*(int*)zv, *(int*)cv, "copy_deep val %zu", i);
    }

    // ---- map / reduce ----
    zarray_map(z, bump);
    array_map(c, bump);
    compare_all(z, c, "map");
    long za = 0, ca = 0;
    zarray_reduce(z, fold, &za);
    array_reduce(c, fold, &ca);
    CHECK_EQ(za, ca, "reduce %ld vs %ld", za, ca);

    // ---- filter (on fresh copies so both start equal) ----
    ZArray* zf = nilptr;
    Array* cf = nilptr;
    CHECK_EQ(zarray_copy_shallow(z, &zf), array_copy_shallow(c, &cf), "filter copy");
    CHECK_EQ(zarray_filter(zf, is_even, &zs), array_filter(cf, is_even, &cs), "filter rc");
    compare_all(zs, cs, "filter");
    CHECK_EQ(zarray_filter_mut(zf, is_even), array_filter_mut(cf, is_even), "filter_mut rc");
    compare_all(zf, cf, "filter_mut");

    // ---- trim ----
    CHECK_EQ(zarray_trim_capacity(z), array_trim_capacity(c), "trim rc");
    CHECK_EQ(zarray_capacity(z), array_capacity(c), "capacity after trim");

    // ---- iterators ----
    ArrayIter zit, cit;
    zarray_iter_init(&zit, z);
    array_iter_init(&cit, c);
    int steps = 0;
    while (1) {
        void *zv = nilptr, *cv = nilptr;
        enum cc_stat rz = zarray_iter_next(&zit, &zv);
        enum cc_stat rc = array_iter_next(&cit, &cv);
        CHECK_EQ(rz, rc, "iter rc step=%d", steps);
        if (rz == CC_ITER_END) break;
        CHECK(zv == cv, "iter val step=%d", steps);
        CHECK_EQ(zarray_iter_index(&zit), array_iter_index(&cit), "iter index");
        if ((steps % 3) == 0) {
            void *zrr = nilptr, *crr = nilptr;
            CHECK_EQ(zarray_iter_remove(&zit, &zrr), array_iter_remove(&cit, &crr), "iter rm rc");
            CHECK(zrr == crr, "iter rm val");
        }
        steps++;
    }
    compare_all(z, c, "iter remove");

    // ---- zip iterators ----
    ArrayZipIter zzi, czi;
    zarray_zip_iter_init(&zzi, z, zs);
    array_zip_iter_init(&czi, c, cs);
    int zsteps = 0;
    while (1) {
        void *z1 = nilptr, *z2 = nilptr, *c1 = nilptr, *c2 = nilptr;
        enum cc_stat rz = zarray_zip_iter_next(&zzi, &z1, &z2);
        enum cc_stat rc = array_zip_iter_next(&czi, &c1, &c2);
        CHECK_EQ(rz, rc, "zip rc step=%d", zsteps);
        if (rz == CC_ITER_END) break;
        CHECK(z1 == c1 && z2 == c2, "zip vals step=%d", zsteps);
        CHECK_EQ(zarray_zip_iter_index(&zzi), array_zip_iter_index(&czi), "zip index");
        zsteps++;
    }

    // ---- buffer / remove_all / destroy ----
    CHECK(zarray_get_buffer(z) != nilptr || zarray_size(z) == 0, "get_buffer");
    zarray_remove_all(z);
    array_remove_all(c);
    compare_all(z, c, "remove_all");

    zarray_destroy(zs);
    array_destroy(cs);
    zarray_destroy(zf);
    array_destroy(cf);
    zarray_destroy(z);
    array_destroy(c);

    t_join_send(done);
}

static void controller(void* arg) {
    (void)arg;
    done = crn_hchan_new(0);
    crn_post(zuser, 0);
    t_join_wait(done, 1);
    t_report_and_exit();
}

int main(void) {
    t_init();
    crn_post(controller, 0);
    for (;;) { sleep(1); }
}
