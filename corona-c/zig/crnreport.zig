// crnreport — small Zig utility that consumes the zigstr package.
//
//   crnreport version
//   crnreport summary <logfile>...
//
// It reads runtests.sh logs and prints a summary table. This is the "real use"
// of the zigstr dependency (String builder for formatting), independent from
// the C runtime.
//
// Zig 0.17 API: main receives std.process.Init (io + gpa + args); file IO goes
// through std.Io.Dir, not std.fs.

const std = @import("std");
const String = @import("zigstr").String;

const version = "0.1.0";

pub fn main(init: std.process.Init) !void {
    const alloc = init.gpa;
    var it = init.minimal.args.iterate();
    _ = it.next(); // argv[0]

    const cmd = it.next() orelse {
        try usage();
        return;
    };

    if (std.mem.eql(u8, cmd, "version")) {
        var s = String.init(alloc);
        defer s.deinit();
        try s.concat("corona-c crnreport ");
        try s.concat(version);
        try s.concat(" (zigstr-backed)");
        std.debug.print("{s}\n", .{s.str()});
        return;
    }

    if (std.mem.eql(u8, cmd, "summary")) {
        var pass: usize = 0;
        var fail: usize = 0;
        var retry: usize = 0;
        var files: usize = 0;

        while (it.next()) |path| {
            const data = std.Io.Dir.cwd().readFileAlloc(
                init.io,
                path,
                alloc,
                .limited(16 * 1024 * 1024),
            ) catch |err| {
                std.debug.print("cannot read {s}: {s}\n", .{ path, @errorName(err) });
                continue;
            };
            defer alloc.free(data);
            files += 1;

            var lines = std.mem.splitScalar(u8, data, '\n');
            while (lines.next()) |line| {
                if (std.mem.indexOf(u8, line, "PASS t") != null) pass += 1;
                if (std.mem.indexOf(u8, line, "FAIL t") != null) fail += 1;
                if (std.mem.indexOf(u8, line, "RETRY t") != null) retry += 1;
            }
        }

        var s = String.init(alloc);
        defer s.deinit();
        try s.concat("logs:  ");
        try concatNum(&s, files);
        try s.concat("\nPASS:  ");
        try concatNum(&s, pass);
        try s.concat("\nFAIL:  ");
        try concatNum(&s, fail);
        try s.concat("\nRETRY: ");
        try concatNum(&s, retry);
        std.debug.print("{s}\n", .{s.str()});
        if (fail > 0) std.process.exit(1);
        return;
    }

    try usage();
}

fn concatNum(s: *String, n: usize) !void {
    var buf: [32]u8 = undefined;
    const txt = try std.fmt.bufPrint(&buf, "{d}", .{n});
    try s.concat(txt);
}

fn usage() !void {
    std.debug.print(
        \\usage:
        \\  crnreport version
        \\  crnreport summary <logfile>...
        \\
    , .{});
}
