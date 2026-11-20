// corona_zig.zig — C-ABI demo functions, compiled into libcrn.
//
// Pure std, no allocator, no libc calls, so it can be built standalone with
// `zig build-obj` from CMake as well as through build.zig.
//
// Compiled with -fPIC: the same object ends up in libcrn.a and libcrn.so.

const std = @import("std");

const version = "0.1.0";

const voidptr = ?*anyopaque;
const coidptr = ?*const anyopaque;

/// Returns a static greeting string.
export fn crn_zig_hello() [*:0]const u8 {
    return "hello from zig";
}

/// Simple integer addition.
export fn crn_zig_add(a: i32, b: i32) i32 {
    return a +% b;
}

/// Iterative fibonacci (u64 wraparound on overflow is not possible for u32 n
/// within reasonable inputs; we saturate at n > 92 to stay in u64 range).
export fn crn_zig_fib(n: u32) u64 {
    if (n > 92) return std.math.maxInt(u64);
    var a: u64 = 0;
    var b: u64 = 1;
    var i: u32 = 0;
    while (i < n) : (i += 1) {
        const t = a +% b;
        a = b;
        b = t;
    }
    return a;
}

/// Sum of an int32 array (len elements), widened to i64.
export fn crn_zig_sum(ptr: [*]const i32, len: usize) i64 {
    var s: i64 = 0;
    var i: usize = 0;
    while (i < len) : (i += 1) {
        s += @as(i64, ptr[i]);
    }
    return s;
}

/// Whether the array is non-decreasing.
export fn crn_zig_is_sorted(ptr: [*]const i32, len: usize) bool {
    if (len < 2) return true;
    var i: usize = 1;
    while (i < len) : (i += 1) {
        if (ptr[i - 1] > ptr[i]) return false;
    }
    return true;
}

/// Writes "corona-zig/<version>" into the caller's buffer (NUL-terminated when
/// it fits) and returns the number of bytes written (excluding the NUL).
export fn crn_zig_banner(buf: [*]u8, cap: usize) usize {
    if (cap == 0) return 0;
    const out = std.fmt.bufPrint(buf[0 .. cap - 1], "corona-zig/{s}", .{version}) catch
        buf[0 .. cap - 1];
    buf[out.len] = 0;
    return out.len;
}

test "add" {
    try std.testing.expectEqual(@as(i32, 5), crn_zig_add(2, 3));
}

test "fib" {
    try std.testing.expectEqual(@as(u64, 55), crn_zig_fib(10));
}

// ===========================================================================
// zarray — collectc array.h semantics implemented over std.ArrayListUnmanaged.
// Exported with C ABI as zarray_* ; signatures mirror collectc's array.h.
// All allocations go through ArrayConf's mem_* callbacks (default: libc).
// ===========================================================================

const CC_OK: c_int = 0;
const CC_ERR_ALLOC: c_int = 1;
const CC_ERR_INVALID_CAPACITY: c_int = 2;
const CC_ERR_INVALID_RANGE: c_int = 3;
const CC_ERR_MAX_CAPACITY: c_int = 4;
const CC_ERR_VALUE_NOT_FOUND: c_int = 7;
const CC_ERR_OUT_OF_RANGE: c_int = 8;
const CC_ITER_END: c_int = 9;

const CC_MAX_ELEMENTS: usize = std.math.maxInt(usize) - 1;
const ZA_DEFAULT_CAPACITY: usize = 8;
const ZA_DEFAULT_EXP: f32 = 2.0;

const AllocFn = *const fn (usize) callconv(.c) voidptr;
const CallocFn = *const fn (usize, usize) callconv(.c) voidptr;
const FreeFn = *const fn (voidptr) callconv(.c) void;
const CmpFn = *const fn (coidptr, coidptr) callconv(.c) c_int;
const PredFn = *const fn (coidptr) callconv(.c) bool;
const MapFn = *const fn (voidptr) callconv(.c) void;
const ReduceFn = *const fn (voidptr, voidptr, voidptr) callconv(.c) void;
const CpFn = *const fn (voidptr) callconv(.c) voidptr;

const ZArrayConf = extern struct {
    capacity: usize,
    exp_factor: f32,
    mem_alloc: ?AllocFn,
    mem_calloc: ?CallocFn,
    mem_free: ?FreeFn,
};

// same layout as collectc's ArrayIter / ArrayZipIter (ar is ZArray* in C terms)
const ZArrayIter = extern struct { ar: voidptr, index: usize, last_removed: bool };
const ZArrayZipIter = extern struct { ar1: voidptr, ar2: voidptr, index: usize, last_removed: bool };

pub const ZArray = struct {
    items: std.ArrayListUnmanaged(voidptr) = .empty,
    exp: f32 = ZA_DEFAULT_EXP,
    am: AllocFn,
    cm: CallocFn,
    fm: FreeFn,
};

// std.Allocator adapter backed by the C callbacks from ArrayConf
const CAlloc = struct {
    am: AllocFn,
    cm: CallocFn,
    fm: FreeFn,

    fn allocator(self: *CAlloc) std.mem.Allocator {
        return .{ .ptr = self, .vtable = &vt };
    }
    const vt: std.mem.Allocator.VTable = .{
        .alloc = aAlloc,
        .resize = aResize,
        .remap = aRemap,
        .free = aFree,
    };
    fn aAlloc(ctx: voidptr, len: usize, al: std.mem.Alignment, ra: usize) ?[*]u8 {
        _ = ra;
        const self: *CAlloc = @ptrCast(@alignCast(ctx));
        if (al.toByteUnits() > 16) return null; // libc/GC provide >= 16
        const p = self.am(len) orelse return null;
        return @ptrCast(p);
    }
    fn aResize(ctx: voidptr, mem: []u8, al: std.mem.Alignment, nl: usize, ra: usize) bool {
        _ = ctx; _ = mem; _ = al; _ = nl; _ = ra;
        return false;
    }
    fn aRemap(ctx: voidptr, mem: []u8, al: std.mem.Alignment, nl: usize, ra: usize) ?[*]u8 {
        _ = ctx; _ = mem; _ = al; _ = nl; _ = ra;
        return null;
    }
    fn aFree(ctx: voidptr, mem: []u8, al: std.mem.Alignment, ra: usize) void {
        _ = al; _ = ra;
        const self: *CAlloc = @ptrCast(@alignCast(ctx));
        self.fm(@ptrCast(mem.ptr));
    }
};

fn zOf(p: voidptr) *ZArray {
    return @ptrCast(@alignCast(p.?));
}
fn zAlloc(z: *ZArray, buf: *CAlloc) std.mem.Allocator {
    buf.* = .{ .am = z.am, .cm = z.cm, .fm = z.fm };
    return buf.allocator();
}

fn zExpand(z: *ZArray, a: std.mem.Allocator) c_int {
    const cap = z.items.capacity;
    if (cap == CC_MAX_ELEMENTS) return CC_ERR_MAX_CAPACITY;
    var ncap: usize = @intFromFloat(@as(f64, @floatFromInt(cap)) * @as(f64, z.exp));
    if (ncap <= cap) ncap = cap + 1;
    if (ncap >= CC_MAX_ELEMENTS) ncap = CC_MAX_ELEMENTS;
    z.items.ensureTotalCapacityPrecise(a, ncap) catch return CC_ERR_ALLOC;
    return CC_OK;
}

export fn zarray_conf_init(conf: ?*ZArrayConf) void {
    const c = conf orelse return;
    c.capacity = ZA_DEFAULT_CAPACITY;
    c.exp_factor = ZA_DEFAULT_EXP;
    c.mem_alloc = null;
    c.mem_calloc = null;
    c.mem_free = null;
}

// declared as plain C externs: the object is also built standalone by CMake's
// `zig build-obj`, where libc is not linked at compile time
extern "c" fn malloc(size: usize) voidptr;
extern "c" fn calloc(blocks: usize, size: usize) voidptr;
extern "c" fn free(ptr: voidptr) void;

fn dAlloc(n: usize) callconv(.c) voidptr {
    return malloc(n);
}
fn dCalloc(n: usize, sz: usize) callconv(.c) voidptr {
    return calloc(n, sz);
}
fn dFree(p: voidptr) callconv(.c) void {
    free(p);
}

export fn zarray_new_conf(conf: ?*const ZArrayConf, out: ?*?*ZArray) c_int {
    const o = out orelse return CC_ERR_ALLOC;
    o.* = null;
    var cfg: ZArrayConf = .{
        .capacity = ZA_DEFAULT_CAPACITY,
        .exp_factor = ZA_DEFAULT_EXP,
        .mem_alloc = null,
        .mem_calloc = null,
        .mem_free = null,
    };
    if (conf) |c| cfg = c.*;
    const am = cfg.mem_alloc orelse &dAlloc;
    const cm = cfg.mem_calloc orelse &dCalloc;
    const fm = cfg.mem_free orelse &dFree;
    var exp = cfg.exp_factor;
    if (!(exp > 1)) exp = ZA_DEFAULT_EXP;
    const cap = cfg.capacity;
    if (cap == 0) return CC_ERR_INVALID_CAPACITY;
    // mirror: exp >= CC_MAX_ELEMENTS / capacity  -> invalid
    if (@as(f64, exp) >= @as(f64, @floatFromInt(CC_MAX_ELEMENTS / cap))) return CC_ERR_INVALID_CAPACITY;

    const raw = cm(1, @sizeOf(ZArray)) orelse return CC_ERR_ALLOC;
    const z: *ZArray = @ptrCast(@alignCast(raw));
    z.* = .{ .items = .empty, .exp = exp, .am = am, .cm = cm, .fm = fm };
    var buf: CAlloc = undefined;
    const a = zAlloc(z, &buf);
    z.items.ensureTotalCapacityPrecise(a, cap) catch {
        fm(raw);
        return CC_ERR_ALLOC;
    };
    o.* = z;
    return CC_OK;
}

export fn zarray_new(out: ?*?*ZArray) c_int {
    return zarray_new_conf(null, out);
}

export fn zarray_destroy(arp: ?*ZArray) void {
    const z = arp orelse return;
    var buf: CAlloc = undefined;
    const a = zAlloc(z, &buf);
    z.items.deinit(a);
    z.fm(@ptrCast(z));
}

export fn zarray_destroy_cb(arp: ?*ZArray, cb: ?MapFn) void {
    const z = arp orelse return;
    if (cb) |f| for (z.items.items) |e| f(e);
    zarray_destroy(z);
}

export fn zarray_add(arp: ?*ZArray, e: voidptr) c_int {
    const z = arp orelse return CC_ERR_ALLOC;
    var buf: CAlloc = undefined;
    const a = zAlloc(z, &buf);
    if (z.items.items.len == CC_MAX_ELEMENTS) return CC_ERR_MAX_CAPACITY;
    if (z.items.items.len == z.items.capacity) {
        const r = zExpand(z, a);
        if (r != CC_OK) return r;
    }
    z.items.appendAssumeCapacity(e);
    return CC_OK;
}

export fn zarray_add_at(arp: ?*ZArray, e: voidptr, idx: usize) c_int {
    const z = arp orelse return CC_ERR_ALLOC;
    var buf: CAlloc = undefined;
    const a = zAlloc(z, &buf);
    const n = z.items.items.len;
    if (idx > n) return CC_ERR_OUT_OF_RANGE;
    if (n == CC_MAX_ELEMENTS) return CC_ERR_MAX_CAPACITY;
    if (n == z.items.capacity) {
        const r = zExpand(z, a);
        if (r != CC_OK) return r;
    }
    z.items.insertAssumeCapacity(idx, e);
    return CC_OK;
}

export fn zarray_replace_at(arp: ?*ZArray, e: voidptr, idx: usize, out: ?*voidptr) c_int {
    const z = arp orelse return CC_ERR_ALLOC;
    if (idx >= z.items.items.len) return CC_ERR_OUT_OF_RANGE;
    if (out) |o| o.* = z.items.items[idx];
    z.items.items[idx] = e;
    return CC_OK;
}

export fn zarray_swap_at(arp: ?*ZArray, index1: usize, index2: usize) c_int {
    const z = arp orelse return CC_ERR_ALLOC;
    const n = z.items.items.len;
    if (index1 >= n or index2 >= n) return CC_ERR_OUT_OF_RANGE;
    const t = z.items.items[index1];
    z.items.items[index1] = z.items.items[index2];
    z.items.items[index2] = t;
    return CC_OK;
}

fn zRemoveAt(z: *ZArray, idx: usize, out: ?*voidptr) c_int {
    const n = z.items.items.len;
    if (idx >= n) return CC_ERR_OUT_OF_RANGE;
    const v = z.items.orderedRemove(idx);
    if (out) |o| o.* = v;
    return CC_OK;
}

export fn zarray_remove_at(arp: ?*ZArray, idx: usize, out: ?*voidptr) c_int {
    const z = arp orelse return CC_ERR_ALLOC;
    return zRemoveAt(z, idx, out);
}

export fn zarray_remove(arp: ?*ZArray, e: voidptr, out: ?*voidptr) c_int {
    const z = arp orelse return CC_ERR_ALLOC;
    for (z.items.items, 0..) |v, i| {
        if (v == e) return zRemoveAt(z, i, out);
    }
    return CC_ERR_VALUE_NOT_FOUND;
}

export fn zarray_remove_last(arp: ?*ZArray, out: ?*voidptr) c_int {
    const z = arp orelse return CC_ERR_ALLOC;
    if (z.items.items.len == 0) return CC_ERR_VALUE_NOT_FOUND;
    const v = z.items.pop().?;
    if (out) |o| o.* = v;
    return CC_OK;
}

export fn zarray_remove_all(arp: ?*ZArray) void {
    const z = arp orelse return;
    z.items.clearRetainingCapacity();
}

export fn zarray_remove_all_free(arp: ?*ZArray) void {
    const z = arp orelse return;
    for (z.items.items) |e| z.fm(e);
    z.items.clearRetainingCapacity();
}

export fn zarray_get_at(arp: ?*ZArray, idx: usize, out: ?*voidptr) c_int {
    const z = arp orelse return CC_ERR_ALLOC;
    const o = out orelse return CC_ERR_OUT_OF_RANGE;
    if (idx >= z.items.items.len) return CC_ERR_OUT_OF_RANGE;
    o.* = z.items.items[idx];
    return CC_OK;
}

export fn zarray_get_last(arp: ?*ZArray, out: ?*voidptr) c_int {
    const z = arp orelse return CC_ERR_ALLOC;
    const o = out orelse return CC_ERR_OUT_OF_RANGE;
    if (z.items.items.len == 0) return CC_ERR_OUT_OF_RANGE;
    o.* = z.items.items[z.items.items.len - 1];
    return CC_OK;
}

export fn zarray_size(arp: ?*ZArray) usize {
    return if (arp) |z| z.items.items.len else 0;
}

export fn zarray_capacity(arp: ?*ZArray) usize {
    return if (arp) |z| z.items.capacity else 0;
}

export fn zarray_get_buffer(arp: ?*ZArray) ?[*]const coidptr {
    const z = arp orelse return null;
    return @ptrCast(z.items.items.ptr);
}

fn zNewLike(src: *ZArray, out: ?*?*ZArray) ?*ZArray {
    const o = out orelse return null;
    o.* = null;
    var conf: ZArrayConf = .{
        .capacity = if (src.items.capacity > 0) src.items.capacity else ZA_DEFAULT_CAPACITY,
        .exp_factor = src.exp,
        .mem_alloc = src.am,
        .mem_calloc = src.cm,
        .mem_free = src.fm,
    };
    var tmp: ?*ZArray = null;
    if (zarray_new_conf(&conf, &tmp) != CC_OK) return null;
    o.* = tmp;
    return tmp;
}

export fn zarray_subarray(arp: ?*ZArray, from: usize, to: usize, out: ?*?*ZArray) c_int {
    const z = arp orelse return CC_ERR_ALLOC;
    if (from > to or to >= z.items.items.len) return CC_ERR_INVALID_RANGE;
    const dst = zNewLike(z, out) orelse return CC_ERR_ALLOC;
    for (z.items.items[from .. to + 1]) |e| {
        const r = zarray_add(dst, e);
        if (r != CC_OK) {
            zarray_destroy(dst);
            return r;
        }
    }
    return CC_OK;
}

export fn zarray_copy_shallow(arp: ?*ZArray, out: ?*?*ZArray) c_int {
    const z = arp orelse return CC_ERR_ALLOC;
    const dst = zNewLike(z, out) orelse return CC_ERR_ALLOC;
    for (z.items.items) |e| {
        const r = zarray_add(dst, e);
        if (r != CC_OK) {
            zarray_destroy(dst);
            return r;
        }
    }
    return CC_OK;
}

export fn zarray_copy_deep(arp: ?*ZArray, cp: ?CpFn, out: ?*?*ZArray) c_int {
    const z = arp orelse return CC_ERR_ALLOC;
    const f = cp orelse return zarray_copy_shallow(arp, out);
    const dst = zNewLike(z, out) orelse return CC_ERR_ALLOC;
    for (z.items.items) |e| {
        const r = zarray_add(dst, f(e));
        if (r != CC_OK) {
            zarray_destroy(dst);
            return r;
        }
    }
    return CC_OK;
}

export fn zarray_reverse_(arp: ?*ZArray) void {
    const z = arp orelse return;
    std.mem.reverse(voidptr, z.items.items);
}

export fn zarray_trim_capacity(arp: ?*ZArray) c_int {
    const z = arp orelse return CC_ERR_ALLOC;
    var buf: CAlloc = undefined;
    const a = zAlloc(z, &buf);
    const n = z.items.items.len;
    if (n == z.items.capacity) return CC_OK;
    z.items.shrinkAndFree(a, n);
    return CC_OK;
}

export fn zarray_contains(arp: ?*ZArray, e: voidptr) usize {
    const z = arp orelse return 0;
    var n: usize = 0;
    for (z.items.items) |v| {
        if (v == e) n += 1;
    }
    return n;
}

export fn zarray_contains_value(arp: ?*ZArray, e: voidptr, cmp: ?CmpFn) usize {
    const z = arp orelse return 0;
    const f = cmp orelse return zarray_contains(arp, e);
    var n: usize = 0;
    var ev: voidptr = e;
    for (z.items.items) |v| {
        var vv: voidptr = v;
        if (f(@ptrCast(&ev), @ptrCast(&vv)) == 0) n += 1;
    }
    return n;
}

export fn zarray_index_of(arp: ?*ZArray, e: voidptr, idx: ?*usize) c_int {
    const z = arp orelse return CC_ERR_ALLOC;
    const o = idx orelse return CC_ERR_VALUE_NOT_FOUND;
    for (z.items.items, 0..) |v, i| {
        if (v == e) {
            o.* = i;
            return CC_OK;
        }
    }
    return CC_ERR_VALUE_NOT_FOUND;
}

const SortCtx = struct { f: CmpFn };
fn zLess(ctx: SortCtx, a: voidptr, b: voidptr) bool {
    var av: voidptr = a;
    var bv: voidptr = b;
    return ctx.f(@ptrCast(&av), @ptrCast(&bv)) < 0;
}

export fn zarray_sort(arp: ?*ZArray, cmp: ?CmpFn) void {
    const z = arp orelse return;
    const f = cmp orelse return;
    std.sort.pdq(voidptr, z.items.items, SortCtx{ .f = f }, zLess);
}

export fn zarray_map(arp: ?*ZArray, fnc: ?MapFn) void {
    const z = arp orelse return;
    if (fnc) |f| for (z.items.items) |e| f(e);
}

export fn zarray_reduce(arp: ?*ZArray, fnc: ?ReduceFn, result: voidptr) void {
    const z = arp orelse return;
    const f = fnc orelse return;
    const n = z.items.items.len;
    if (n == 0) return;
    if (n == 1) {
        f(z.items.items[0], null, result);
        return;
    }
    f(z.items.items[0], z.items.items[1], result);
    var i: usize = 2;
    while (i < n) : (i += 1) {
        f(result, z.items.items[i], result);
    }
}

export fn zarray_filter_mut(arp: ?*ZArray, pred: ?PredFn) c_int {
    const z = arp orelse return CC_ERR_ALLOC;
    const p = pred orelse return CC_OK;
    var w: usize = 0;
    for (z.items.items) |e| {
        if (p(e)) {
            z.items.items[w] = e;
            w += 1;
        }
    }
    z.items.shrinkRetainingCapacity(w);
    return CC_OK;
}

export fn zarray_filter(arp: ?*ZArray, pred: ?PredFn, out: ?*?*ZArray) c_int {
    const z = arp orelse return CC_ERR_ALLOC;
    const p = pred orelse return zarray_copy_shallow(arp, out);
    const dst = zNewLike(z, out) orelse return CC_ERR_ALLOC;
    for (z.items.items) |e| {
        if (!p(e)) continue;
        const r = zarray_add(dst, e);
        if (r != CC_OK) {
            zarray_destroy(dst);
            return r;
        }
    }
    return CC_OK;
}

// ---- iterators (collectc semantics) ----
export fn zarray_iter_init(iter: ?*ZArrayIter, arp: ?*ZArray) void {
    const it = iter orelse return;
    it.ar = arp;
    it.index = 0;
    it.last_removed = false;
}

export fn zarray_iter_next(iter: ?*ZArrayIter, out: ?*voidptr) c_int {
    const it = iter orelse return CC_ITER_END;
    const z = zOf(it.ar);
    if (it.index >= z.items.items.len) return CC_ITER_END;
    if (out) |o| o.* = z.items.items[it.index];
    it.index += 1;
    it.last_removed = false;
    return CC_OK;
}

export fn zarray_iter_remove(iter: ?*ZArrayIter, out: ?*voidptr) c_int {
    const it = iter orelse return CC_ERR_OUT_OF_RANGE;
    if (it.last_removed) return CC_ERR_VALUE_NOT_FOUND;
    if (it.index == 0) return CC_ERR_OUT_OF_RANGE;
    const r = zRemoveAt(zOf(it.ar), it.index - 1, out);
    if (r == CC_OK) {
        // cltc keeps iter->index unchanged after a removal
        it.last_removed = true;
    }
    return r;
}

export fn zarray_iter_add(iter: ?*ZArrayIter, e: voidptr) c_int {
    const it = iter orelse return CC_ERR_OUT_OF_RANGE;
    const r = zarray_add_at(zOf(it.ar), e, it.index);
    if (r == CC_OK) it.index += 1;
    return r;
}

export fn zarray_iter_replace(iter: ?*ZArrayIter, e: voidptr, out: ?*voidptr) c_int {
    const it = iter orelse return CC_ERR_OUT_OF_RANGE;
    if (it.index == 0) return CC_ERR_OUT_OF_RANGE;
    return zarray_replace_at(zOf(it.ar), e, it.index - 1, out);
}

export fn zarray_iter_index(iter: ?*ZArrayIter) usize {
    // cltc: return iter->index - 1
    return if (iter) |it| it.index -% 1 else 0;
}

export fn zarray_zip_iter_init(iter: ?*ZArrayZipIter, a1: ?*ZArray, a2: ?*ZArray) void {
    const it = iter orelse return;
    it.ar1 = a1;
    it.ar2 = a2;
    it.index = 0;
    it.last_removed = false;
}

export fn zarray_zip_iter_next(iter: ?*ZArrayZipIter, o1: ?*voidptr, o2: ?*voidptr) c_int {
    const it = iter orelse return CC_ITER_END;
    const z1 = zOf(it.ar1);
    const z2 = zOf(it.ar2);
    if (it.index >= z1.items.items.len or it.index >= z2.items.items.len) return CC_ITER_END;
    if (o1) |o| o.* = z1.items.items[it.index];
    if (o2) |o| o.* = z2.items.items[it.index];
    it.index += 1;
    it.last_removed = false;
    return CC_OK;
}

export fn zarray_zip_iter_add(iter: ?*ZArrayZipIter, e1: voidptr, e2: voidptr) c_int {
    const it = iter orelse return CC_ERR_OUT_OF_RANGE;
    const i = it.index;
    const r1 = zarray_add_at(zOf(it.ar1), e1, i);
    if (r1 != CC_OK) return r1;
    const r2 = zarray_add_at(zOf(it.ar2), e2, i);
    if (r2 != CC_OK) return r2;
    it.index += 1;
    return CC_OK;
}

export fn zarray_zip_iter_remove(iter: ?*ZArrayZipIter, o1: ?*voidptr, o2: ?*voidptr) c_int {
    const it = iter orelse return CC_ERR_OUT_OF_RANGE;
    if (it.index == 0) return CC_ERR_OUT_OF_RANGE;
    if (it.last_removed) return CC_ERR_VALUE_NOT_FOUND;
    const z1 = zOf(it.ar1);
    const z2 = zOf(it.ar2);
    const i = it.index - 1;
    if (i >= z1.items.items.len or i >= z2.items.items.len) return CC_ERR_OUT_OF_RANGE;
    const r1 = zRemoveAt(z1, i, o1);
    const r2 = zRemoveAt(z2, i, o2);
    if (r1 == CC_OK and r2 == CC_OK) {
        // cltc keeps iter->index unchanged after a removal
        it.last_removed = true;
        return CC_OK;
    }
    return if (r1 != CC_OK) r1 else r2;
}

export fn zarray_zip_iter_replace(iter: ?*ZArrayZipIter, e1: voidptr, e2: voidptr, o1: ?*voidptr, o2: ?*voidptr) c_int {
    const it = iter orelse return CC_ERR_OUT_OF_RANGE;
    if (it.index == 0) return CC_ERR_OUT_OF_RANGE;
    const i = it.index - 1;
    const r1 = zarray_replace_at(zOf(it.ar1), e1, i, o1);
    const r2 = zarray_replace_at(zOf(it.ar2), e2, i, o2);
    return if (r1 != CC_OK) r1 else r2;
}

export fn zarray_zip_iter_index(iter: ?*ZArrayZipIter) usize {
    // cltc: return iter->index - 1
    return if (iter) |it| it.index -% 1 else 0;
}
