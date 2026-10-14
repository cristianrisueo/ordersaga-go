# Imagen de un servicio de OrderSaga: compila cmd/${SERVICE} y lo ejecuta en una imagen mínima

#* Compilación

# Etapa de compilación con la versión de Go del proyecto; SERVICE elige el binario (orders, stock o payments)
FROM golang:1.27.2-alpine AS build
ARG SERVICE
WORKDIR /src

# Descarga las dependencias primero: esta capa se reutiliza mientras no cambien go.mod y go.sum
COPY go.mod go.sum ./
RUN go mod download

# Copia el código y compila un binario estático del servicio
COPY . .
RUN CGO_ENABLED=0 go build -o /app/service ./cmd/${SERVICE}

#* Ejecución

# Etapa final: alpine trae wget para el healthcheck
FROM alpine:3

# Usuario sin privilegios para ejecutar el servicio
RUN adduser -D -H app

# Solo el binario; el modo (api, producer) lo pone compose como command
COPY --from=build /app/service /app/service
USER app
EXPOSE 8000
ENTRYPOINT ["/app/service"]
