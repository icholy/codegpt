# CodeGPT

> Pipe STDIN through GPT-4

Usage:

```
# use in a pipe
cat file.go | codegpt "switch from testify to gotest.tools assertions"

# update a file in place
$ codegpt -w -s test.spec.js "convert to chai assertions"
```

Vim Usage:

```
:%!codegpt "add doc comments"
```
