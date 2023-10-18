FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o temporary-namespaces

FROM scratch
COPY --from=builder /app/temporary-namespaces /
# TODO: metrics
# EXPOSE 8080
CMD ["/temporary-namespaces"]
