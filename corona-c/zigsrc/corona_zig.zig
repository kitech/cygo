// corona_zig.zig — C-ABI demo functions, compiled into libcrn.
//
// Pure std, no allocator, no libc calls, so it can be built standalone with
// `zig build-obj` from CMake as well as through build.zig.
//
// Compiled with -fPIC: the same object ends up in libcrn.a and libcrn.so.

const std = @import("std");

const version = "0.1.0";

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
