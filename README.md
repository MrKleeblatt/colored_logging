# Colored Logging

A logging library for the go programming language. Features:
- colored output in 6 different colors: FATAL, ERROR, WARN, INFO, DEBUG, TRACE
- print file, function and line number, where ERROR, FATAL, DEBUG and TRACE were called
- print call stack trace on TRACE
- pipe print output (without colors) to log file
- should be thread safe

See `log_test.go`