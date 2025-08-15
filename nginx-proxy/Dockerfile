FROM openresty/openresty:alpine

# Install dependencies and configure
RUN apk add --no-cache ca-certificates curl perl build-base wget tar yaml-dev git && \
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
    luarocks install lua-resty-redis

# Copy configuration files
COPY nginx.conf /usr/local/openresty/nginx/conf/nginx.conf
COPY cache_metrics.conf /usr/local/openresty/nginx/conf/cache_metrics.conf
COPY providers.json /app/providers.json
COPY cache_rules.yaml /app/cache_rules.yaml
COPY lua /usr/local/openresty/nginx/lua

# Specify the port to be used
EXPOSE 8080

# Nginx startup command
CMD ["openresty", "-g", "daemon off;"]
