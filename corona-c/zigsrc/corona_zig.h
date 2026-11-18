// corona_zig.h — C declarations for the Zig C-ABI demo functions.
// Implemented in zigsrc/corona_zig.zig and linked into libcrn (static/shared).

#ifndef CORONA_ZIG_H
#define CORONA_ZIG_H

#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

// static greeting, NUL-terminated
const char* crn_zig_hello(void);

int32_t crn_zig_add(int32_t a, int32_t b);

uint64_t crn_zig_fib(uint32_t n);

int64_t crn_zig_sum(const int32_t* ptr, size_t len);

bool crn_zig_is_sorted(const int32_t* ptr, size_t len);

// Writes "corona-zig/<version>" into buf (NUL-terminated when it fits).
// Returns bytes written, excluding the NUL.
size_t crn_zig_banner(char* buf, size_t cap);

#ifdef __cplusplus
}
#endif

#endif
