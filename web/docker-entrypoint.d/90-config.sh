#!/usr/bin/env sh
set -eu

CERT_FILE="${TLS_CERT_FILE:-}"
KEY_FILE="${TLS_KEY_FILE:-}"

if [ -n "$CERT_FILE" ] && [ -n "$KEY_FILE" ]; then
  cat >/etc/nginx/conf.d/default.conf <<EOF
server {
  listen 80;
  server_name _;
  return 301 https://\$host\$request_uri;
}

server {
  listen 443 ssl;
  server_name _;
  root /usr/share/nginx/html;

  ssl_certificate /certs/${CERT_FILE};
  ssl_certificate_key /certs/${KEY_FILE};

  location / {
    try_files \$uri /index.html;
  }

  location /api/ {
    proxy_pass http://server:8080;
    proxy_http_version 1.1;
    proxy_set_header Host \$host;
    proxy_set_header X-Real-IP \$remote_addr;
    proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto \$scheme;
  }
}
EOF
else
  if [ -n "$CERT_FILE" ] || [ -n "$KEY_FILE" ]; then
    echo "TLS_CERT_FILE or TLS_KEY_FILE missing, TLS disabled." >&2
  fi
  cat >/etc/nginx/conf.d/default.conf <<'EOF'
server {
  listen 80;
  server_name _;
  root /usr/share/nginx/html;

  location / {
    try_files $uri /index.html;
  }

  location /api/ {
    proxy_pass http://server:8080;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
  }
}
EOF
fi
