# Algolia Search logs

Show the last entries of the Algolia Search API's [logs](https://www.algolia.com/doc/rest-api/search/#tag/Advanced/operation/getLogs) endpoint.

This utility is written in Go and distributed as source code.
To use it on your computer, clone the repository and run `go build` inside its directory.

You'll need a configuration file in your home directory (`~/.config/search-logs.env`) with the following content:

```sh
ALGOLIA_APPLICATION_ID=...
ALGOLIA_API_KEY=...
```

To show the last 10 log entries, run:

```sh
./search-logs
```

To show the available command-line options, run:

```sh
./search-logs -h
```
