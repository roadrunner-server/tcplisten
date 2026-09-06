<p align="center">
 <a href="https://pkg.go.dev/github.com/roadrunner-server/tcplisten?tab=doc"><img src="https://godoc.org/github.com/roadrunner-server/tcplisten?status.svg"></a>
	<a href="https://github.com/roadrunner-server/tcplisten/actions"><img src="https://github.com/roadrunner-server/tcplisten/workflows/tests/badge.svg" alt=""></a>
	<a href="https://goreportcard.com/report/github.com/roadrunner-server/tcplisten"><img src="https://goreportcard.com/badge/github.com/roadrunner-server/tcplisten"></a>
	<a href="https://codecov.io/gh/roadrunner-server/tcplisten/"><img src="https://codecov.io/gh/roadrunner-server/tcplisten/branch/master/graph/badge.svg"></a>
	<a href="https://discord.gg/TFeEmCs"><img src="https://img.shields.io/badge/discord-chat-magenta.svg"></a>
</p>

Package tcplisten provides a customizable TCP `net.Listener` with various
performance-related options:

 * `SO_REUSEPORT`: This option allows linear scaling of server performance
   on multi-CPU servers.
   See https://www.nginx.com/blog/socket-sharding-nginx-release-1-9-1/ for details.

 * `TCP_DEFER_ACCEPT`: This option expects the server read from the accepted
   connection before writing to them.

 * `TCP_FASTOPEN`: See https://lwn.net/Articles/508865/ for details.


[Documentation](https://godoc.org/github.com/roadrunner-server/tcplisten).

## UNIX Socket Attributes

`CreateListener(address)` keeps the operating system defaults for socket permissions and ownership. Use `CreateListenerWithOptions` to set attributes on a filesystem UNIX socket:

```go
gid := 33
listener, err := tcplisten.CreateListenerWithOptions(
    "unix:///run/roadrunner/fcgi.sock",
    &tcplisten.UnixSocketOptions{Mode: "0660", GID: &gid},
)
```

- `Mode` must be an octal string from `"0000"` through `"0777"`. An empty string keeps the default mode.
- `UID` and `GID` are optional numeric IDs. Nil keeps the default. Zero is a valid ID. Each ID must fit in an `int` and be less than `4294967295`.
- A nil options pointer preserves the existing address behavior. A non-nil pointer, including an empty options struct, requires a filesystem UNIX socket on Linux, macOS, or FreeBSD.
- TCP, Linux abstract sockets, and Windows reject non-nil options. Windows can still create listeners without these options.
- `Validate(address)` checks options without filesystem access. A nil receiver is valid.
- The caller must close the returned listener. The library closes it if an attribute change fails.

The library sets pathname ownership before mode. It does not change the process umask, worker credentials, or parent directories. The process must have permission to perform the requested operations. An unprivileged owner can change the group only to one of its groups.

Attributes are set after the socket starts listening. A client can connect before these changes finish. Successful return guarantees the requested final attributes, not access control during setup. Existing directory permissions and umask must restrict initial access. Parent directories must prevent untrusted path replacement throughout the listener lifetime. Clients also need search permission on every parent directory.

An existing filesystem socket is removed before bind. Regular files, directories, and symlinks are not removed. Socket replacement does not check whether another process still uses the socket. On Linux, addresses that start with `@` or NUL use the abstract namespace and cause no filesystem changes.

On macOS and FreeBSD, a relative name such as `@socket` is a filesystem path. Go leaves this pathname after a successful listener close. Use `unix://./@socket` for automatic removal, or remove the pathname after closing the listener. On Windows, leading `@` and NUL retain Go's address encoding without filesystem operations; this does not add abstract-socket support to Windows.

The package is derived from [tcplisten](https://github.com/valyala/tcplisten) with modifications.
