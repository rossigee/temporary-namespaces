FROM golang:1.25 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o temporary-namespaces

FROM scratch
COPY --from=builder /app/temporary-namespaces /
EXPOSE 8080
CMD ["/temporary-namespaces"]
