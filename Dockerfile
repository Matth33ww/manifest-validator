# Dockerfile for the Go server.
# It has two stages. The first one builds the program. The second one only runs it.
# So the final image is small and has no Go tools inside.

# Stage 1: build the program.
FROM golang:1.22-alpine AS build
WORKDIR /src

# Copy go.mod first. Docker can reuse this step when only the code changes.
COPY go.mod ./
COPY *.go ./

# Run the tests during the build. If a test fails, the image is not made.
RUN go test ./...

# CGO_ENABLED=0 makes one file with no outside libraries, so it runs on a tiny image.
RUN CGO_ENABLED=0 go build -o /manifest-validator .

# Stage 2: run the program.
# This image is very small. It has HTTPS certificates, which the health check needs.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /manifest-validator /manifest-validator

# The server listens on this port. You can change it with the PORT variable.
ENV PORT=8080
EXPOSE 8080

# Run as a normal user, not as root.
USER nonroot
ENTRYPOINT ["/manifest-validator"]
