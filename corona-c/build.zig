// corona-c Zig build — mirrors CMakeLists.txt + cxrt.cmake
//
// Steps:
//   zig build                 -> crn lib + corona exe + crnreport tool
//   zig build corona          -> just the corona executable
//   zig build tests           -> 14 test executables
//   zig build check           -> run all tests (see tests/runtests.sh for retries)
//   zig build check-str       -> vendored zigstr's own tests
//   zig build bdwgc-zig       -> compile-only bdwgc-zig bindings (never referenced)
//
// Options:
//   -Dnetpoller=libevent|epoll (default libevent, same as CMake)
//   -Doptimize=Debug|ReleaseFast|... (default Debug)
//   -Dtarget=... (default native)

const std = @import("std");

const cflags: []const []const u8 = &.{
    "-std=c11",
    "-D_GNU_SOURCE",
    "-DNRDEBUG",
    "-DCORO_STACKALLOC",
    "-DCORO_UCONTEXT",
    "-DHAVE_UCONTEXT_H",
    "-DHAVE_SETJMP_H",
    "-DHAVE_SIGALTSTACK",
    "-DGC_THREADS",
    "-DLOG_USE_COLOR",
    "-D_XOPEN_SOURCE",
    "-fno-omit-frame-pointer",
    "-fstack-protector-all",
    "-g",
};

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

const test_names: []const []const u8 = &.{
    "t01_spawn", "t02_pingpong", "t03_buffered", "t04_close",
    "t05_select", "t06_timer",   "t07_mutex",    "t08_net",
    "t09_stress", "t10_hammer",  "t11_gcstack",  "t12_sched",
    "t13_handshake", "t14_stats",
};

pub fn build(b: *std.Build) void {
    const target = b.standardTargetOptions(.{});
    const optimize = b.standardOptimizeOption(.{ .preferred_optimize_mode = .Debug });
    const backend = b.option([]const u8, "netpoller", "netpoller backend: libevent|epoll") orelse "libevent";

    const use_epoll = std.mem.eql(u8, backend, "epoll");
    if (!use_epoll and !std.mem.eql(u8, backend, "libevent")) {
        std.debug.panic("unknown -Dnetpoller={s} (use libevent|epoll)", .{backend});
    }

    // ---------------------------------------------------------------- crn lib
    const crn = b.addLibrary(.{
        .name = "crn",
        .linkage = .static,
        .root_module = b.createModule(.{
            .target = target,
            .optimize = optimize,
            .link_libc = true,
        }),
    });

    // corona-c sources: the two netpoller backends define the same symbols,
    // so exactly one of them is compiled in (mirrors cxrt.cmake).
    for (corona_srcs) |f| crn.root_module.addCSourceFile(.{ .file = b.path(f), .flags = cflags });
    crn.root_module.addCSourceFile(.{
        .file = b.path(if (use_epoll) netpoller_epoll else netpoller_libevent),
        .flags = cflags,
    });

    crn.root_module.addCSourceFiles(.{
        .root = b.path("../3rdparty/cltc/src"),
        .files = cltc_srcs,
        .flags = cflags,
    });
    crn.root_module.addCSourceFile(.{ .file = b.path("../src/creflect.c"), .flags = cflags });

    crn.root_module.addIncludePath(b.path("."));
    crn.root_module.addIncludePath(b.path("../src"));
    crn.root_module.addIncludePath(b.path("../include"));
    crn.root_module.addIncludePath(b.path("../3rdparty/cltc/src"));
    crn.root_module.addIncludePath(b.path("../3rdparty/cltc/src/include"));
    crn.root_module.addIncludePath(b.path("../3rdparty"));

    b.installArtifact(crn);

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
    exe.rdynamic = true; // replaces -Wl,--export-dynamic (PLT hooks)

    linkRuntimeLibs(exe);

    b.installArtifact(exe);
    const run_corona = b.addRunArtifact(exe);
    const run_step = b.step("run", "Run the corona demo");
    run_step.dependOn(&run_corona.step);

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
        t.root_module.addIncludePath(b.path("."));
        t.root_module.addIncludePath(b.path("../src"));
        t.root_module.addIncludePath(b.path("../3rdparty/cltc/src"));
        t.root_module.addIncludePath(b.path("../3rdparty/cltc/src/include"));
        t.root_module.addIncludePath(b.path("../3rdparty"));
        t.root_module.linkLibrary(crn);
        linkRuntimeLibs(t);

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

fn linkRuntimeLibs(c: *std.Build.Step.Compile) void {
    for ([_][]const u8{ "gc", "event", "event_pthreads", "sigsegv", "dl", "pthread" }) |l| {
        c.root_module.linkSystemLibrary(l, .{});
    }
}
