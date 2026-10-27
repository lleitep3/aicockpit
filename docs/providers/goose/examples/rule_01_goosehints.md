# .goosehints
## Project: Rust CLI Tool
- Always use `clap` v4 for argument parsing.
- Use `thiserror` and `anyhow` for error handling.
- When writing tests, place them in a `tests` module at the bottom of the file using `#[cfg(test)]`.
- Run `cargo fmt` and `cargo clippy` after every significant logic change.
