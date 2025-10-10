FROM ubuntu:20.04

RUN apt-get update && \
    apt-get install -y g++ libhiredis-dev libpq-dev libssl-dev ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY . .

RUN g++ -std=c++17 -I/usr/include/postgresql/ main.cpp -o judge_service -lhiredis -lhiredis_ssl -lpq -lssl -lcrypto -lstdc++fs

CMD ["./judge_service"]

