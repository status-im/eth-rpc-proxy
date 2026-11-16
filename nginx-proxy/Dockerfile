# Build context: the repository root (proxy-common is a submodule there):
#   docker build -f nginx-proxy/Dockerfile .
FROM openresty/openresty:alpine

# Install dependencies and configure
RUN apk add --no-cache ca-certificates curl perl build-base wget tar yaml-dev zlib-dev git && \
    update-ca-certificates && \
    /usr/local/openresty/bin/opm install ledgetech/lua-resty-http && \
    cd /tmp && \
    wget https://luarocks.org/releases/luarocks-3.12.2.tar.gz && \
    tar -xzf luarocks-3.12.2.tar.gz && \
    cd luarocks-3.12.2 && \
    ./configure --prefix=/usr/local --with-lua=/usr/local/openresty/luajit && \
    make && make install && \
    cd / && rm -rf /tmp/luarocks-3.12.2* && \
    luarocks install lyaml && \
    luarocks install lua-resty-mlcache && \
    luarocks install lua-resty-redis && \
    luarocks install lua-zlib

# Copy configuration files
COPY nginx-proxy/nginx.conf /usr/local/openresty/nginx/conf/nginx.conf
COPY nginx-proxy/metrics.conf /usr/local/openresty/nginx/conf/metrics.conf
# cache_metrics.conf removed - metrics now served by go-proxy-cache service
COPY nginx-proxy/providers.json /app/providers.json
# Puzzle auth for nginx, shared with nft-proxy and market-proxy; it also
# brings utils/resolver_utils.lua, which the provider loader uses
COPY proxy-common/auth/nginx/conf/auth_locations.conf /usr/local/openresty/nginx/conf/auth_locations.conf
COPY proxy-common/auth/nginx/lua /usr/local/openresty/nginx/lua
COPY nginx-proxy/lua /usr/local/openresty/nginx/lua

# Specify the port to be used
EXPOSE 8080

# Nginx startup command
CMD ["openresty", "-g", "daemon off;"]
