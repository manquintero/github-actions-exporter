FROM golang:1.18 as builder

WORKDIR /app
COPY . .
RUN make build


FROM alpine:latest as release

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
COPY --from=builder /app/bin/app .
CMD ["./app"]
