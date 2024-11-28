#!/bin/bash

# Exit on any error
set -e

# Function to log messages
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    log "Please run as root or with sudo"
    exit 1
fi

# Create alerim user if it doesn't exist
if ! id "alerim" &>/dev/null; then
    log "Creating alerim user..."
    useradd -m -s /bin/bash alerim
fi

# Install required dependencies
log "Installing dependencies..."
apt-get update
apt-get install -y \
    build-essential \
    git \
    curl \
    nginx \
    software-properties-common \
    ufw \
    fail2ban \
    certbot \
    python3-certbot-nginx

# Install Go
if ! command_exists go; then
    log "Installing Go..."
    wget https://go.dev/dl/go1.17.linux-amd64.tar.gz
    tar -C /usr/local -xzf go1.17.linux-amd64.tar.gz
    echo "export PATH=$PATH:/usr/local/go/bin" >> /home/alerim/.bashrc
    rm go1.17.linux-amd64.tar.gz
fi

# Configure firewall
log "Configuring firewall..."
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh
ufw allow 80/tcp
ufw allow 443/tcp
ufw allow 8545/tcp
ufw allow 9000/tcp
ufw --force enable

# Configure fail2ban
log "Configuring fail2ban..."
cat > /etc/fail2ban/jail.local << EOF
[DEFAULT]
bantime = 3600
findtime = 600
maxretry = 5

[sshd]
enabled = true

[nginx-http-auth]
enabled = true
EOF

systemctl restart fail2ban

# Clone and build Alerim
log "Setting up Alerim..."
cd /home/alerim
git clone https://github.com/alexandrut83/alerimAIM.git alerim
cd alerim
chown -R alerim:alerim .
sudo -u alerim make build

# Setup blockchain node service
log "Setting up blockchain node service..."
cp bin/alerimnode /usr/local/bin/
chmod 755 /usr/local/bin/alerimnode

cat > /etc/systemd/system/alerim-node.service << EOF
[Unit]
Description=Alerim Blockchain Node
After=network.target

[Service]
Type=simple
User=alerim
ExecStart=/usr/local/bin/alerimnode
Restart=always
RestartSec=3
LimitNOFILE=4096

[Install]
WantedBy=multi-user.target
EOF

# Configure Nginx
log "Configuring Nginx..."
cat > /etc/nginx/sites-available/alerim << EOF
server {
    listen 80;
    listen [::]:80;
    server_name _;  # Accept any domain name

    access_log /var/log/nginx/alerim.access.log;
    error_log /var/log/nginx/alerim.error.log;

    root /var/www/alerim;
    index index.html;

    # Enable directory listing
    autoindex on;

    location / {
        try_files \$uri \$uri/ /index.html;
        add_header 'Access-Control-Allow-Origin' '*';
        add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS';
        add_header 'Access-Control-Allow-Headers' 'DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range';
    }

    location /api {
        proxy_pass http://127.0.0.1:8545;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_cache_bypass \$http_upgrade;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        
        # CORS headers
        add_header 'Access-Control-Allow-Origin' '*';
        add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS';
        add_header 'Access-Control-Allow-Headers' 'DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range';
    }
}
EOF

# Create web directory if it doesn't exist
mkdir -p /var/www/alerim

# Copy web files
cp -r wallet/web/* /var/www/alerim/
chown -R www-data:www-data /var/www/alerim
chmod -R 755 /var/www/alerim

# Enable the site and restart Nginx
ln -sf /etc/nginx/sites-available/alerim /etc/nginx/sites-enabled/
rm -f /etc/nginx/sites-enabled/default

# Test Nginx configuration
nginx -t

# Restart Nginx
systemctl restart nginx

# Deploy web wallet
log "Deploying web wallet..."

# Start services
log "Starting services..."
systemctl daemon-reload
systemctl enable alerim-node
systemctl start alerim-node

# Setup SSL if domain is provided
if [ ! -z "$DOMAIN" ]; then
    log "Setting up SSL for $DOMAIN..."
    certbot --nginx -d "$DOMAIN" --non-interactive --agree-tos --email admin@"$DOMAIN"
fi

log "Deployment complete!"
log "Please configure your domain DNS to point to this server's IP address"
log "Then access your wallet at: http://your_domain.com"
log "For security, please:"
log "1. Change the alerim user password"
log "2. Configure SSL if not done automatically"
log "3. Regularly update system packages"
log "4. Monitor system logs for suspicious activity"
