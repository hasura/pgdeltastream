# Code of Conduct
This project and everyone participating in it is governed by the [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. 

# Development Environment
- Set up PostgreSQL as described in the [README](README.md)
- Install Go 1.10 and set up the Go environment variables
- Install [dep](https://github.com/golang/dep)
- Clone this repo to `$GOPATH/src/github.com/hasura/pgdeltastream`
- Run `dep ensure` to update the dependencies

# Build
Build the project using the `go build` command:

```bash
$ cd $GOPATH/src/github.com/hasura/pgdeltastream
$ go build
```

# Run
Run the application with the arguments specifying database and server parameters: 

```bash
$ ./pgdeltastream dbName pgUser pgHost pgPort serverHost serverPort 
```

# Tests
Tests have been written using the Go testing framework. To run tests make sure you have Postgres running at `localhost:5432`.

Use `go test` to run tests. Example:

```bash
$ cd server
$ go test
<output truncated>
PASS
ok   github.com/hasura/pgdeltastream/server 0.113s
```