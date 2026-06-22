#!/bin/bash
set -e

# 依次对 nonblocking 下所有子项目执行 bins → full
TARGETS=("cockroach" "etcd" "grpc" "istio" "kubernetes" "moby" "serving")
RESULT_DIR="zgortResult"

mkdir -p "$RESULT_DIR"

for name in "${TARGETS[@]}"; do
    echo "============================================"
    echo "=== [$name] bins ==="
    echo "============================================"
    ./bin/fuzz --task bins --path "testdata/gobench/nonblocking/$name" -o "testbins/$name"

    echo "============================================"
    echo "=== [$name] full ==="
    echo "============================================"
    ./bin/fuzz --task full --path "testbins/$name" > "$RESULT_DIR/$name.txt"

    echo "=== [$name] done ==="
    echo ""
done

echo "All done. Results saved to $RESULT_DIR/"
