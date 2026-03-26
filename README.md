# mog

A WebAssembly module orchestration engine. Modules are written in [WAT](https://developer.mozilla.org/en-US/docs/WebAssembly/Understanding_the_text_format), compiled to WASM, and executed via [wazero](https://wazero.io/).

## Prerequisites

- Go 1.25+
- [`wat2wasm`](https://github.com/WebAssembly/wabt) — required to compile WAT source to WASM at runtime

## Running

```sh
go run .
```

## Testing

```sh
go test ./...
```

Tests that depend on `wat2wasm` are skipped automatically when the binary is not on `PATH`.

To run a specific package:

```sh
go test ./module/...
go test ./engine/...
```
