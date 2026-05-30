FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY cmd/mesh/go.mod cmd/mesh/go.sum ./
RUN go mod download
COPY cmd/mesh/ ./
RUN CGO_ENABLED=0 go build -o /arcana-mesh .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-mesh /arcana-mesh
EXPOSE 8083
ENTRYPOINT ["/arcana-mesh"]
