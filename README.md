# filestream [![GoDoc](https://godoc.org/github.com/jaddr2line/filestream?status.svg)](https://godoc.org/github.com/jaddr2line/filestream) [![Build Status](https://travis-ci.org/jaddr2line/filestream.svg?branch=master)](https://travis-ci.org/jaddr2line/filestream)
A system for efficiently streaming bundles of files over a connection.

Features:
* Compression - supports gzip and lz4
* Chunked - can stream files without knowing the size in advance (e.g. generated files/downloads)

## When would I use this?
This package was developed based on poor experiences with [docker's usage of tar](https://godoc.org/github.com/docker/docker/client#Client.CopyToContainer) as a part of their API.
The tar file format requires lengths to be written before the contents of the file.
This created problems, as I had to buffer huge files in memory as they were collected over HTTP or generated on the fly.

I created this for the purpose of streaming files to test machines in a CI system.
This is also potentially useful for data import/export.

## Try it out on the command line
A command line interface has been created as a debugging tool and a demo.
To install it, run:
```
go get github.com/jaddr2line/filestream/cmd/filestream/...
```

To encode files to stdout:
```
# Encode directories "example" and "test" to stdout
filestream example test | somecommand

# Encode to stdout with gzip compression
filestream -z gzip -l 9 example | somecommand
```

Decoding from stdin:
```
# Decode stream to current directory.
curl https://example.com/something | filestream -d

# Decode stream relative to directory example
curl https://example.com/something | filestream -d -C example
```

Listing files in stream:
```
curl https://example.com/something | filestream -t
```

The tool can also read/write the stream to different places:
```
# Store directory example to a file called "output.dat"
filestream example -s output.dat

# Decode from a file
filestream -d -s output.dat

# Decode data from an HTTP GET request
filestream -d -s https://example.com/something
```
