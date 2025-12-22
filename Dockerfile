# Stage 1: Build UI
FROM node:20-alpine AS ui-builder

WORKDIR /app/ui

# Copy UI package files
COPY ui/package*.json ./

# Install dependencies
RUN npm ci

# Copy UI source
COPY ui/ ./

# Build UI
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.24.11-trixie AS go-builder

WORKDIR /app

# Copy go mod files
COPY go.mod ./
COPY go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY main.go ./

# Build Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o xivstrings .

# Stage 3: Final image
FROM gcr.io/distroless/static-debian13

WORKDIR /app

# Copy Go binary from builder
COPY --from=go-builder /app/xivstrings .

# Copy UI dist from builder
COPY --from=ui-builder /app/ui/dist ./ui/dist

VOLUME /app/data

# Expose port
EXPOSE 8080

# Run the application
CMD ["./xivstrings", "-addr", ":8080"]

