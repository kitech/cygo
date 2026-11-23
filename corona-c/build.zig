// corona-c Zig build — mirrors CMakeLists.txt + cxrt.cmake
//
// Steps:
//   zig build                 -> crn libs (static+shared) + cxrt + corona exe + crnreport
//   zig build corona          -> just the corona executable
//   zig build tests           -> 14 test executables
//   zig build check           -> run all tests (see tests/runtests.sh for retries)
//   zig build check-str       -> vendored zigstr's own tests
//   zig build bdwgc-zig       -> compile-only bdwgc-zig bindings (never referenced)
//
// Options:
//   -Dnetpoller=libevent|epoll (default libevent, same as CMake)
//   -Doptimize=Debug|ReleaseFast|... (default Debug)
//   -Dcdebug=true|false (default false: no -g, no stack protector -> fastest C compile)
//   -Dtarget=... (default native)

const std = @import("std");

const cflags_base: []const []const u8 = &.{
    "-std=c11",
    "-D_GNU_SOURCE",
    "-DNRDEBUG",
    "-DCORO_STACKALLOC",
    "-DHAVE_SETJMP_H",
    "-DHAVE_SIGALTSTACK",
    "-DGC_THREADS",
    "-DLOG_USE_COLOR",
    "-O0", // explicit: no optimization, compile-speed first
    "-fno-omit-frame-pointer",
    "-pipe", // stream through pipes instead of temp files
    // no -g: DWARF is the dominant C compile cost (use -Dcdebug=true)
    // no -fstack-protector-all: per-function instrumentation, slower to compile
};

// _XOPEN_SOURCE=600 exists for macOS (explicit level -> correct sizeof(ucontext_t)).
// On the BSDs it clears __BSD_VISIBLE, which hides MSG_DONTWAIT/MSG_NOSIGNAL/
// F_DUPFD_CLOEXEC and the u_char family, so leave the strict XSI level to
// Darwin/Linux.  BSD then sees its normal default namespace.
fn makeCflags(alloc: std.mem.Allocator, cdebug: bool, is_macos: bool, is_android: bool, is_bsd: bool) []const []const u8 {
    var list: std.ArrayList([]const u8) = .empty;
    list.appendSlice(alloc, cflags_base) catch @panic("OOM");
    if (!is_bsd) {
        list.append(alloc, "-D_XOPEN_SOURCE=600") catch @panic("OOM");
    }
    // macOS: _XOPEN_SOURCE hides BSD types (u_char/u_int) used by nameser.h
    if (is_macos) list.append(alloc, "-D_DARWIN_C_SOURCE") catch @panic("OOM");
    // bionic: _XOPEN_SOURCE alone hides some declarations; enable default APIs
    if (is_android) {
        list.append(alloc, "-D_DEFAULT_SOURCE") catch @panic("OOM");
        // bionic has no ucontext; use vendored higan-emu/libco
        list.append(alloc, "-DCORO_LIBCO89") catch @panic("OOM");
        list.append(alloc, "-DLIBCO_MP") catch @panic("OOM");
    } else {
        list.append(alloc, "-DCORO_UCONTEXT") catch @panic("OOM");
        list.append(alloc, "-DHAVE_UCONTEXT_H") catch @panic("OOM");
    }
    if (cdebug) {
        list.append(alloc, "-g") catch @panic("OOM");
        list.append(alloc, "-fstack-protector-strong") catch @panic("OOM");
    }
    return list.toOwnedSlice(alloc) catch @panic("OOM");
}

const corona_srcs: []const []const u8 = &.{
    "coro.c",
    "corowp.c",
    "iohook.c",
    "iohookcb.c",
    "iohook2.c",
    "futex.c",
    "corona_util.c",
    "rxilog.c",
    "atomic.c",
    "datstu.c",
    "szqueue.c",
    "chan.c",
    "hchan.c",
    "hselect.c",
    "coronagc.c",
    "corona.c",
    "functrace.c",
};

const netpoller_libevent = "netpoller_event.c";
const netpoller_epoll = "netpoller_epoll.c";

const cltc_srcs: []const []const u8 = &.{
    "array.c",    "hashtable.c", "treetable.c", "queue.c",
    "deque.c",    "pqueue.c",    "common.c",    "hashset.c",
};

// mirrors add_library(cxrt STATIC ...) in cxrt.cmake (callable.c/cxutil.c are
// deliberately not part of it there either)
const cxrt_srcs: []const []const u8 = &.{
    "cxrtbase.c", "cxmemory.c",  "cxstring.c", "cxhashtable.c",
    "cxarray.c",  "cxqueue.c",   "cxiface.c",  "creflect.c",
};

const test_names: []const []const u8 = &.{
    "t01_spawn", "t02_pingpong", "t03_buffered", "t04_close",
    "t05_select", "t06_timer",   "t07_mutex",    "t08_net",
    "t09_stress", "t10_hammer",  "t11_gcstack",  "t12_sched",
    "t13_handshake", "t14_stats", "t15_zigdemo", "t16_zarray",
};

pub fn build(b: *std.Build) void {
    const target = b.standardTargetOptions(.{});
    const optimize = b.standardOptimizeOption(.{ .preferred_optimize_mode = .Debug });
    const backend = b.option([]const u8, "netpoller", "netpoller backend: libevent|epoll") orelse "libevent";
    const cdebug = b.option(bool, "cdebug", "emit -g and stack protector for C sources (default off: compile speed)") orelse false;
    const is_android = target.result.abi == .android;
    const is_bsd = switch (target.result.os.tag) {
        .freebsd, .netbsd, .openbsd => true,
        else => false,
    };
    // libexecinfo: bionic has no backtrace(), musl has neither <execinfo.h>
    // nor backtrace() (Alpine/Void ship them in libexecinfo). Default on for
    // Android only: zig's bundled musl headers have no <execinfo.h>, so a musl
    // target would fail to link even with the library installed. Native musl
    // builds (CMake) detect it with find_library(); zig users opt in with
    // -Dexecinfo=true when they have a matching libexecinfo for the target.
    const want_execinfo = b.option(bool, "execinfo",
        "link libexecinfo for backtrace() (default: android only; pass true on musl with libexecinfo installed)") orelse is_android;
    const cflags = makeCflags(b.allocator, cdebug, target.result.os.tag == .macos, is_android, is_bsd);

    const use_epoll = std.mem.eql(u8, backend, "epoll");
    if (!use_epoll and !std.mem.eql(u8, backend, "libevent")) {
        std.debug.panic("unknown -Dnetpoller={s} (use libevent|epoll)", .{backend});
    }

    // ------------------------------------------------------ zig C-ABI demos
    // zigsrc/corona_zig.zig: compiled once, the object goes into both libcrn
    // variants (mirrors cxrt.cmake's zig build-obj rule).
    const zig_demo = b.addObject(.{
        .name = "crn_zig_demo",
        .root_module = b.createModule(.{
            .root_source_file = b.path("zigsrc/corona_zig.zig"),
            .target = target,
            .optimize = optimize,
            .pic = true,
            .link_libc = true, // zarray default allocators use malloc/calloc/free
        }),
    });

    // ---------------------------------------------------------------- crn lib
    const backend_file = if (use_epoll) netpoller_epoll else netpoller_libevent;

    // crn_st: mirrors add_library(crn_st STATIC $<TARGET_OBJECTS:crn_objlib>)
    const crn = b.addLibrary(.{
        .name = "crn",
        .linkage = .static,
        .root_module = b.createModule(.{
            .target = target,
            .optimize = optimize,
            .link_libc = true,
        }),
    });
    addCrnSources(b, crn, cflags, backend_file, is_android);
    crn.root_module.addObjectFile(zig_demo.getEmittedBin());
    b.installArtifact(crn);

    // --------------------------------------------------------------- cxrt lib
    // mirrors: add_library(cxrt STATIC src/cxrtbase.c ...) in cxrt.cmake
    const cxrt = b.addLibrary(.{
        .name = "cxrt",
        .linkage = .static,
        .root_module = b.createModule(.{
            .target = target,
            .optimize = optimize,
            .link_libc = true,
        }),
    });
    cxrt.root_module.addCSourceFiles(.{
        .root = b.path("../src"),
        .files = cxrt_srcs,
        .flags = cflags,
    });
    cxrt.root_module.addIncludePath(b.path("."));
    cxrt.root_module.addIncludePath(b.path("../src"));
    cxrt.root_module.addIncludePath(b.path("../include"));
    cxrt.root_module.addIncludePath(b.path("../3rdparty/cltc/src"));
    cxrt.root_module.addIncludePath(b.path("../3rdparty/cltc/src/include"));
    cxrt.root_module.addIncludePath(b.path("../3rdparty"));
    b.installArtifact(cxrt);

    // crn_sh: mirrors add_library(crn_sh SHARED ...) -> libcrn.so
    // PIC comes from .pic = true (CMake used POSITION_INDEPENDENT_CODE 1)
    const crn_sh = b.addLibrary(.{
        .name = "crn",
        .linkage = .dynamic,
        .root_module = b.createModule(.{
            .target = target,
            .optimize = optimize,
            .link_libc = true,
            .pic = true,
        }),
    });
    addCrnSources(b, crn_sh, cflags, backend_file, is_android);
    crn_sh.root_module.addObjectFile(zig_demo.getEmittedBin());
    crn_sh.root_module.linkLibrary(cxrt); // CMake: target_link_libraries(crn_sh cxrt)
    linkRuntimeLibs(crn_sh, want_execinfo); // + ${LIBGC} ${LIBEVENT1} ${LIBEVENT2} ${LIBSIGSEGV}
    b.installArtifact(crn_sh);

    // ------------------------------------------------------------ corona exe
    const exe = b.addExecutable(.{
        .name = "corona",
        .root_module = b.createModule(.{
            .target = target,
            .optimize = optimize,
            .link_libc = true,
        }),
    });
    exe.root_module.addCSourceFile(.{ .file = b.path("main.c"), .flags = cflags });
    exe.root_module.addIncludePath(b.path("."));
    exe.root_module.addIncludePath(b.path("../src"));
    exe.root_module.addIncludePath(b.path("../3rdparty/cltc/src"));
    exe.root_module.addIncludePath(b.path("../3rdparty/cltc/src/include"));
    exe.root_module.addIncludePath(b.path("../3rdparty"));
    exe.root_module.linkLibrary(crn);
    exe.root_module.linkLibrary(cxrt); // CMake: link crn cxrt ${LIBGC}
    exe.rdynamic = true; // replaces -Wl,--export-dynamic (PLT hooks)

    linkRuntimeLibs(exe, want_execinfo);

    b.installArtifact(exe);
    const corona_step = b.step("corona", "Build the corona demo");
    corona_step.dependOn(&b.addInstallArtifact(exe, .{}).step);
    const run_corona = b.addRunArtifact(exe);
    const run_step = b.step("run", "Run the corona demo");
    run_step.dependOn(&run_corona.step);

    // ------------------------------------------------------ main2 soak driver
    const exe2 = b.addExecutable(.{
        .name = "corona2",
        .root_module = b.createModule(.{
            .target = target,
            .optimize = optimize,
            .link_libc = true,
        }),
    });
    exe2.root_module.addCSourceFile(.{ .file = b.path("main2.c"), .flags = cflags });
    exe2.root_module.addIncludePath(b.path("."));
    exe2.root_module.addIncludePath(b.path("../src"));
    exe2.root_module.addIncludePath(b.path("../3rdparty/cltc/src"));
    exe2.root_module.addIncludePath(b.path("../3rdparty/cltc/src/include"));
    exe2.root_module.addIncludePath(b.path("../3rdparty"));
    exe2.root_module.linkLibrary(crn);
    exe2.root_module.linkLibrary(cxrt);
    exe2.rdynamic = true;
    linkRuntimeLibs(exe2, want_execinfo);
    b.installArtifact(exe2);
    const corona2_build_step = b.step("corona2", "Build the main2 soak driver");
    corona2_build_step.dependOn(&b.addInstallArtifact(exe2, .{}).step);
    const run2 = b.addRunArtifact(exe2);
    const run2_step = b.step("run2", "Run the main2 soak driver");
    run2_step.dependOn(&run2.step);

    // ------------------------------------------------------- 14 test programs
    const test_step = b.step("tests", "Build all corona-c tests");
    const check_step = b.step("check", "Build and run all corona-c tests");

    for (test_names) |name| {
        const t = b.addExecutable(.{
            .name = name,
            .root_module = b.createModule(.{
                .target = target,
                .optimize = optimize,
                .link_libc = true,
            }),
        });
        t.root_module.addCSourceFile(.{
            .file = b.path(b.fmt("tests/{s}.c", .{name})),
            .flags = cflags,
        });
        t.root_module.addIncludePath(b.path("tests"));
        t.root_module.addIncludePath(b.path("zigsrc"));
        t.root_module.addIncludePath(b.path("."));
        t.root_module.addIncludePath(b.path("../src"));
        t.root_module.addIncludePath(b.path("../3rdparty/cltc/src"));
        t.root_module.addIncludePath(b.path("../3rdparty/cltc/src/include"));
        t.root_module.addIncludePath(b.path("../3rdparty"));
        t.root_module.linkLibrary(crn);
        t.root_module.linkLibrary(cxrt);
        linkRuntimeLibs(t, want_execinfo);

        test_step.dependOn(&t.step);
        const run_t = b.addRunArtifact(t);
        // tests are long-running; per-test failures are reported by the runner
        check_step.dependOn(&run_t.step);
    }

    // -------------------------------------------------------- zigstr consumer
    const zigstr = b.dependency("zigstr", .{ .target = target, .optimize = optimize });
    const report = b.addExecutable(.{
        .name = "crnreport",
        .root_module = b.createModule(.{
            .root_source_file = b.path("zig/crnreport.zig"),
            .target = target,
            .optimize = optimize,
        }),
    });
    report.root_module.addImport("zigstr", zigstr.module("string"));
    b.installArtifact(report);

    const check_str = b.step("check-str", "Run vendored zigstr tests");
    const str_tests = zigstr.builder.addTest(.{ .root_module = zigstr.module("string") });
    check_str.dependOn(&zigstr.builder.addRunArtifact(str_tests).step);

    // ------------------------------------------------- bdwgc-zig (compile-only)
    // Deliberately NOT part of the default build and never imported/linked.
    // This proves the package can be consumed by this project's toolchain.
    if (b.lazyDependency("bdwgc_zig", .{ .target = target, .optimize = optimize })) |gc_zig| {
        const probe = b.addObject(.{
            .name = "bdwgc_zig_probe",
            .root_module = gc_zig.module("bdwgc"),
        });
        const st = b.step("bdwgc-zig", "Compile bdwgc-zig bindings only (not referenced)");
        st.dependOn(&probe.step);
    }
}

fn addCrnSources(
    b: *std.Build,
    c: *std.Build.Step.Compile,
    cflags: []const []const u8,
    backend_file: []const u8,
    is_android: bool,
) void {
    // corona-c sources: the two netpoller backends define the same symbols,
    // so exactly one of them is compiled in (mirrors cxrt.cmake).
    for (corona_srcs) |f| c.root_module.addCSourceFile(.{ .file = b.path(f), .flags = cflags });
    c.root_module.addCSourceFile(.{ .file = b.path(backend_file), .flags = cflags });
    if (is_android) {
        // bionic has no ucontext; vendored higan-emu/libco
        // repo:   https://github.com/higan-emu/libco
        // commit: e18e09d634d612a01781168ad4d76be10a7e3bad (2024-09-09, v20, ISC)
        c.root_module.addCSourceFile(.{ .file = b.path("../3rdparty/libco/libco.c"), .flags = cflags });
    }
    c.root_module.addCSourceFiles(.{
        .root = b.path("../3rdparty/cltc/src"),
        .files = cltc_srcs,
        .flags = cflags,
    });
    // creflect.c lives in cxrt (same as CMake), not here

    c.root_module.addIncludePath(b.path("."));
    c.root_module.addIncludePath(b.path("../src"));
    c.root_module.addIncludePath(b.path("../include"));
    c.root_module.addIncludePath(b.path("../3rdparty/cltc/src"));
    c.root_module.addIncludePath(b.path("../3rdparty/cltc/src/include"));
    c.root_module.addIncludePath(b.path("../3rdparty"));
}

fn linkRuntimeLibs(c: *std.Build.Step.Compile, want_execinfo: bool) void {
    for ([_][]const u8{ "gc", "event", "event_pthreads", "sigsegv", "dl", "pthread" }) |l| {
        c.root_module.linkSystemLibrary(l, .{});
    }
    // bionic/musl: backtrace()/backtrace_symbols_fd() live in libexecinfo,
    // not libc (see the __has_include guard in src/cxrtbase.c and main2.c).
    if (want_execinfo) c.root_module.linkSystemLibrary("execinfo", .{});
}
