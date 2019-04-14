# filestream [![GoDoc](https://godoc.org/github.com/jadr2ddude/filestream?status.svg)](https://godoc.org/github.com/jadr2ddude/filestream) [![Build Status](https://travis-ci.org/jadr2ddude/filestream.svg?branch=master)](https://travis-ci.org/jadr2ddude/filestream)
A system for efficiently streaming bundles of files over a connection.

Features:
* Compression - supports gzip and lz4
* Chunked - can stream files without knowing the size in advance (e.g. generated files/downloads)
