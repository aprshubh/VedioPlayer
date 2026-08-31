FROM alpine:3.19

WORKDIR /app
COPY server /app/server
RUN chmod +x /app/server

EXPOSE 8080
CMD ["/app/server"]
