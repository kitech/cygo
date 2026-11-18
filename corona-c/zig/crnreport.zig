// crnreport — small Zig utility that consumes the zigstr package.
//
//   crnreport version
//   crnreport summary <logfile>...
//
// It reads runtests.sh logs and prints a summary table. This is the "real use"
// of the zigstr dependency (String builder for formatting), independent from
// the C runtime.

const std = @import("std");
const String = @import("zigstr").String;

const version = "0.1.0";

pub fn main() !void {
    var da: std.heap.DebugAllocator(.{}) = .init;
    defer _ = da.deinit();
    const alloc = da.allocator();

    const args = try std.process.argsAlloc(alloc);
    defer std.process.argsFree(alloc, args);

    if (args.len < 2) {
        try usage();
        return;
    }

    if (std.mem.eql(u8, args[1], "version")) {
        var s = String.init(alloc);
        defer s.deinit();
        try s.concat("corona-c crnreport ");
        try s.concat(version);
        try s.concat(" (zigstr-backed)");
        std.debug.print("{s}\n", .{s.str()});
        return;
    }

    if (std.mem.eql(u8, args[1], "summary")) {
        var pass: usize = 0;
        var fail: usize = 0;
        var retry: usize = 0;
        var files: usize = 0;

        for (args[2..]) |path| {
            const data = std.fs.cwd().readFileAlloc(alloc, path, 16 * 1024 * 1024) catch |err| {
                std.debug.print("cannot read {s}: {s}\n", .{ path, @errorName(err) });
                continue;
            };
            defer alloc.free(data);
            files += 1;

            var it = std.mem.splitScalar(u8, data, '\n');
            while (it.next()) |line| {
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
