FROM openresty/openresty:alpine

# Install dependencies and configure
RUN apk add --no-cache ca-certificates curl perl && \
    update-ca-certificates && \
    /usr/local/openresty/bin/opm install ledgetech/lua-resty-http

# Copy configuration files
COPY nginx.conf /usr/local/openresty/nginx/conf/nginx.conf
COPY cache_metrics.conf /usr/local/openresty/nginx/conf/cache_metrics.conf
COPY providers.json /usr/local/openresty/nginx/providers.json
COPY lua /usr/local/openresty/nginx/lua

# Specify the port to be used
EXPOSE 8080

# Nginx startup command
CMD ["openresty", "-g", "daemon off;"]
