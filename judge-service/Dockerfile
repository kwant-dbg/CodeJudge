FROM ubuntu:20.04

RUN apt-get update && \
    apt-get install -y g++ libhiredis-dev libpq-dev

WORKDIR /app

COPY . .

RUN g++ -std=c++17 -I/usr/include/postgresql/ main.cpp -o judge_service -lhiredis -lpq

CMD ["./judge_service"]

