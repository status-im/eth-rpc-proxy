#!/bin/sh

if [ -z "$1" ] || [ -z "$2" ]; then
    echo "Usage: $0 <path_to_nginx_conf> <cors_origin_url>"
    echo "Example: $0 /etc/nginx/nginx.conf http://localhost:3000"
    exit 1
fi

NGINX_CONF="$1"
CORS_ORIGIN="$2"

# Create a backup of the original file
cp "$NGINX_CONF" "${NGINX_CONF}.bak"

# Function to add CORS configuration to a location block
add_cors_config() {
    local file="$1"
    local location="$2"
    
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
    awk -v loc="$location" -v origin="$CORS_ORIGIN" '
        $0 ~ "location " loc " {" {
            print
            print "            # CORS headers for regular requests"
            print "            add_header '\''Access-Control-Allow-Origin'\'' '\''" origin "'\'' always;"
            print "            add_header '\''Access-Control-Allow-Methods'\'' '\''GET, POST, OPTIONS'\'' always;"
            print "            add_header '\''Access-Control-Allow-Headers'\'' '\''DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization,If-None-Match,Accept-Encoding'\'' always;"
            print "            add_header '\''Access-Control-Expose-Headers'\'' '\''Content-Length,Content-Range,X-Proxy-Cache,X-Response-Size,ETag,Content-Encoding,Vary'\'' always;"
            print ""
            print "            # Handle OPTIONS method"
            print "            if (\$request_method = '\''OPTIONS'\'') {"
            print "                add_header '\''Access-Control-Allow-Origin'\'' '\''" origin "'\'' always;"
            print "                add_header '\''Access-Control-Allow-Methods'\'' '\''GET, POST, OPTIONS'\'' always;"
            print "                add_header '\''Access-Control-Allow-Headers'\'' '\''DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization,If-None-Match,Accept-Encoding'\'' always;"
            print "                add_header '\''Access-Control-Expose-Headers'\'' '\''Content-Length,Content-Range,X-Proxy-Cache,X-Response-Size,ETag,Content-Encoding,Vary'\'' always;"
            print "                add_header '\''Access-Control-Max-Age'\'' 1728000;"
            print "                add_header '\''Content-Type'\'' '\''text/plain; charset=utf-8'\'';"
            print "                return 204;"
            print "            }"
            next
        }
        { print }
    ' "$temp_file" > "$new_file"
    
    # Replace the original file with the new one
    mv "$new_file" "$file"
    rm "$temp_file"
}



# Add CORS configuration for all endpoints
add_cors_config "$NGINX_CONF" "\/auth\/"
add_cors_config "$NGINX_CONF" "\/"

echo "Added CORS configuration to $NGINX_CONF for RPC proxy with origin $CORS_ORIGIN"
