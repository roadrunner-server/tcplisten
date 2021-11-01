//go:build darwin
// +build darwin

package tcplisten

var newSocketCloexec = newSocketCloexecOld
