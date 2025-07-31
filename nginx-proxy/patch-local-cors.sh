#!/usr/bin/env sh

if [ -z "$1" ] || [ -z "$2" ]; then
    echo "Usage: $0 <path_to_nginx_conf> <cors_origin_url>"
    echo "Example: $0 /etc/nginx/nginx.conf http://localhost:3000"
    exit 1
fi

NGINX_CONF="$1"
CORS_ORIGIN="$2"

# Create a backup of the original file
cp "$NGINX_CONF" "${NGINX_CONF}.bak"

# Function to generate CORS configuration block
generate_cors_config() {
    local origin="$1"
    local indent="$2"
    
    cat << EOF
${indent}# CORS headers for regular requests
${indent}add_header 'Access-Control-Allow-Origin' '${origin}' always;
${indent}add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS' always;
${indent}add_header 'Access-Control-Allow-Headers' 'DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization,If-None-Match,Accept-Encoding' always;
${indent}add_header 'Access-Control-Expose-Headers' 'Content-Length,Content-Range,X-Proxy-Cache,X-Response-Size,ETag,Content-Encoding,Vary' always;

${indent}# Handle OPTIONS method
${indent}if (\$request_method = 'OPTIONS') {
${indent}    add_header 'Access-Control-Allow-Origin' '${origin}' always;
${indent}    add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS' always;
${indent}    add_header 'Access-Control-Allow-Headers' 'DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization,If-None-Match,Accept-Encoding' always;
${indent}    add_header 'Access-Control-Expose-Headers' 'Content-Length,Content-Range,X-Proxy-Cache,X-Response-Size,ETag,Content-Encoding,Vary' always;
${indent}    add_header 'Access-Control-Max-Age' 1728000;
${indent}    add_header 'Content-Type' 'text/plain; charset=utf-8';
${indent}    return 204;
${indent}}
EOF
}

# Function to add CORS configuration to a location block
add_cors_config() {
    local file="$1"
    local location="$2"
    
    # Generate CORS configuration
    local cors_config=$(generate_cors_config "$CORS_ORIGIN" "            ")
    
    # Create a temporary file with the CORS configuration
    local temp_file=$(mktemp)

    # Extract the location block
    awk -v loc="$location" '
        $0 ~ "location (= )?" loc " {" { p=1; print; next }
        p==1 && $0 ~ "}" { p=0; print; next }
        p==1 { print; next }
        { print }
    ' "$file" > "$temp_file"
    
    # Create a new file with the CORS headers inserted
    local new_file=$(mktemp)
    awk -v loc="$location" -v cors_config="$cors_config" '
        $0 ~ "location " loc " {" {
            print
            print cors_config
            next
        }
        { print }
    ' "$temp_file" > "$new_file"
    
    # Replace the original file with the new one
    mv "$new_file" "$file"
    rm "$temp_file"
}

# Function to patch cache_metrics.conf - comment out allow lines and add CORS for local development
patch_cache_metrics_config() {
    local nginx_conf_path="$1"
    local cors_origin="$2"
    
    # Get the directory where nginx.conf is located
    local nginx_dir=$(dirname "$nginx_conf_path")
    local cache_metrics_conf="${nginx_dir}/cache_metrics.conf"
    
    # Check if cache_metrics.conf exists
    if [ ! -f "$cache_metrics_conf" ]; then
        echo "Warning: cache_metrics.conf not found in $nginx_dir"
        return 0
    fi
    
    # Create a backup of the original file
    cp "$cache_metrics_conf" "${cache_metrics_conf}.bak"
    
    # Comment out the allow line for local development
    sed -i.tmp 's/^[[:space:]]*allow[[:space:]]/    # allow /' "$cache_metrics_conf"
    
    # Generate CORS configuration
    local cors_config=$(generate_cors_config "$cors_origin" "    ")
    
    # Add CORS headers after access_log off; line
    awk -v cors_config="$cors_config" '
        /access_log off;/ {
            print
            print ""
            print cors_config
            next
        }
        { print }
    ' "$cache_metrics_conf" > "${cache_metrics_conf}.tmp"
    
    # Replace the original file with the modified one
    mv "${cache_metrics_conf}.tmp" "$cache_metrics_conf"
    
    echo "Patched $cache_metrics_conf - commented out allow restrictions and added CORS headers for local development"
}

# Add CORS configuration for all endpoints
add_cors_config "$NGINX_CONF" "\/auth\/"
add_cors_config "$NGINX_CONF" "\/"

# Patch cache_metrics.conf
patch_cache_metrics_config "$NGINX_CONF" "$CORS_ORIGIN"

echo "Added CORS configuration to $NGINX_CONF for RPC proxy with origin $CORS_ORIGIN"
