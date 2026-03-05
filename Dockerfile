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

# Copy go mod files first
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

COPY main.go ./
COPY pkg/ ./pkg/

# Build Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o xivstrings .

# Stage 3: Final image
FROM gcr.io/distroless/static-debian13

WORKDIR /app

# Copy Go binary from builder
COPY --from=go-builder /app/xivstrings .

# Copy UI dist from builder
COPY --from=ui-builder /app/ui/dist ./ui/dist

# Persistent data: version file, strings/<version>/, index/<version>/
# On first run the server will fetch latest ixion release and populate this.
VOLUME /app/data

# Optional: set to allow POST /api/version?token=... to trigger updates.
# If unset, POST /api/version returns 403.
# ENV XIVSTRINGS_UPDATE_TOKEN=

# Expose port
EXPOSE 8080

# -data /app/data: use volume for version, strings, and index
CMD ["./xivstrings", "-addr", ":8080", "-data", "/app/data"]

