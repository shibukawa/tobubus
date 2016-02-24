// General Purpose Plugin System TobuBus
// by Yoshiki Shibukawa under MIT

// Package tobubus is a general purpose plug-in system. It is designed for working with Golang and Qt.
// Both Golang and Qt can work as hosts and plugins.
//
// Design
//
// Originally it started as plug-in system, but now, it is smaller version of d-bus or anything.
//
// 1. Host and Plugins are run as process.
//
// 2. Unix Domain Socket are used on Linux/BSD/Mac OS, Named pipe on Windows(compatible with Qt's QLocalSocket/QLocalServer)
//
// 3. Host and plugins publish their own objects to the name space (it has URL-ish address)
//
// 4. Host and plugins call others' object's method (dRuby inspired)
//
// 5. It supports bi-direction method call
package tobubus
