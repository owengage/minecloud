#!/bin/bash
set -ex
cargo build --release --target x86_64-unknown-linux-musl
cp target/x86_64-unknown-linux-musl/release/region_render bootstrap
zip lambda.zip bootstrap
