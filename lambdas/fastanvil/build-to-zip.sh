#!/bin/bash
set -ex
cargo build --release --target x86_64-unknown-linux-musl
cp target/x86_64-unknown-linux-musl/release/fastanvil bootstrap
zip lambda.zip bootstrap
