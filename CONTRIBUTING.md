These are welcome. To get you started, the code has some documentation, a godoc
page, at least one implementation of each interface and tests.

Comments line lengths should be limited to 80 characters wide. Try not to make
source code lines too long. More lines is fine with the exception of
declarations of exported identifiers; they should be on one line, otherwise the
generated godoc looks weird.

Run `go vet` on the source. The GitHub Actions also check for this.
```
just vet
```

There are also tests, those should pass.

One of the goals of this project is to minimize the amount of dependencies
outside of the standard library. The preference is to only add another
dependency when it's specific to a database driver.

Code and tests for any driver live at `drivers/<name_of_driver>`. Any driver is
expected to behave as specified by the `godfish.Driver` interface. Those tests
live at `internal/test`.

The GitHub Actions run a security scanner on all of the source code using
[gosec](https://github.com/securego/gosec). There should be no rule violations
here. The Justfile provides a convenience target if you want to run `gosec` on
your local machine.
