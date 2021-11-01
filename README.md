<p align="center">
 <a href="https://pkg.go.dev/github.com/spiral/tcplisten?tab=doc"><img src="https://godoc.org/github.com/spiral/tcplisten?status.svg"></a>
	<a href="https://github.com/spiral/tcplisten/actions"><img src="https://github.com/spiral/tcplisten/workflows/tests/badge.svg" alt=""></a>
	<a href="https://goreportcard.com/report/github.com/spiral/tcplisten"><img src="https://goreportcard.com/badge/github.com/spiral/tcplisten"></a>
	<a href="https://codecov.io/gh/spiral/tcplisten/"><img src="https://codecov.io/gh/spiral/tcplisten/branch/master/graph/badge.svg"></a>
	<a href="https://lgtm.com/projects/g/spiral/tcplisten/alerts/"><img alt="Total alerts" src="https://img.shields.io/lgtm/alerts/g/spiral/tcplisten.svg?logo=lgtm&logoWidth=18"/></a>
	<a href="https://discord.gg/TFeEmCs"><img src="https://img.shields.io/badge/discord-chat-magenta.svg"></a>
</p>

Package tcplisten provides customizable TCP net.Listener with various
performance-related options:

 * SO_REUSEPORT. This option allows linear scaling server performance
   on multi-CPU servers.
   See https://www.nginx.com/blog/socket-sharding-nginx-release-1-9-1/ for details.

 * TCP_DEFER_ACCEPT. This option expects the server reads from the accepted
   connection before writing to them.

 * TCP_FASTOPEN. See https://lwn.net/Articles/508865/ for details.


[Documentation](https://godoc.org/github.com/spiral/tcplisten).

The package is derived from [tcplisten](https://github.com/valyala/tcplisten).
