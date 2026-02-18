FROM node:22-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.23-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -o /upal ./cmd/upal

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=backend /upal /upal
COPY --from=backend /app/web/dist /web/dist
EXPOSE 8080
ENTRYPOINT ["/upal", "serve"]
