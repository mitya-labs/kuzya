# Сборка llama.cpp под Poco F1 (SD845: Cortex-A75/A55, armv8.2-a + fp16; dotprod и i8mm у этого SoC НЕТ (нет asimddp в /proc/cpuinfo)).
# Базовый образ — Debian sid: на телефоне glibc 2.43, бинарник не должен требовать новее.
FROM --platform=linux/arm64 debian:sid-slim AS build
RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential cmake git ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /src
RUN git clone --depth 1 https://github.com/ggml-org/llama.cpp . && git rev-parse HEAD > /commit.txt
RUN cmake -B build \
        -DCMAKE_BUILD_TYPE=Release \
        -DBUILD_SHARED_LIBS=OFF \
        -DGGML_NATIVE=OFF \
        -DGGML_CPU_ARM_ARCH=armv8.2-a+fp16 \
        -DGGML_OPENMP=OFF \
        -DLLAMA_CURL=OFF \
        -DLLAMA_BUILD_TESTS=OFF \
    && cmake --build build -j"$(nproc)" --target llama-server llama-cli llama-bench

FROM scratch AS out
COPY --from=build /src/build/bin/llama-server /src/build/bin/llama-cli /src/build/bin/llama-bench /commit.txt /
