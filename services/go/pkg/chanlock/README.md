# chanlock

This is a fork of _some aspects of_ [go-deadlock](https://github.com/sasha-s/go-deadlock), a library for diagnosing locking issues with mutexes. It repurposes the deadlock detection mechanism for use with Go code that uses the `for { select { ... } }` pattern.
