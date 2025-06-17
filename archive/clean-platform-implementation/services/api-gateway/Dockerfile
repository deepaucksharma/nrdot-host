FROM nginx:1.24-alpine

# Install dependencies
RUN apk add --no-cache curl

# Copy nginx configuration
COPY nginx.conf /etc/nginx/nginx.conf
COPY default.conf /etc/nginx/conf.d/default.conf

# Create non-root user
RUN adduser -D -H -u 1000 -s /sbin/nologin nginx-user

# Create necessary directories
RUN mkdir -p /var/cache/nginx /var/log/nginx /etc/nginx/conf.d \
    && chown -R nginx-user:nginx-user /var/cache/nginx /var/log/nginx /etc/nginx/conf.d \
    && touch /var/run/nginx.pid \
    && chown nginx-user:nginx-user /var/run/nginx.pid

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

USER nginx-user

EXPOSE 8080

CMD ["nginx", "-g", "daemon off;"]