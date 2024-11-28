# Deploying Alerim on Ubuntu

This guide explains how to deploy the Alerim cryptocurrency node and web wallet on Ubuntu.

## Prerequisites

1. Ubuntu Server 20.04 LTS or later
2. Root or sudo access
3. Open ports:
   - 80 (HTTP)
   - 8545 (Node API)
   - 9000 (P2P Network)

## Installation Steps

1. Update system packages:
```bash
sudo apt-get update
sudo apt-get upgrade -y
```

2. Install required dependencies:
```bash
sudo apt-get install -y \
    build-essential \
    git \
    curl \
    nginx \
    software-properties-common \
    ufw
```

3. Install Go:
```bash
wget https://go.dev/dl/go1.17.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.17.linux-amd64.tar.gz
echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.bashrc
source ~/.bashrc
```

4. Configure firewall:
```bash
sudo ufw allow 80/tcp
sudo ufw allow 8545/tcp
sudo ufw allow 9000/tcp
sudo ufw enable
```

5. Clone Alerim repository:
```bash
git clone https://github.com/yourusername/alerim.git
cd alerim
```

6. Build the node:
```bash
make build
```

7. Create system service:
```bash
sudo tee /etc/systemd/system/alerim-node.service << EOF
[Unit]
Description=Alerim Blockchain Node
After=network.target

[Service]
Type=simple
User=alerim
ExecStart=/usr/local/bin/alerim-node
Restart=always
RestartSec=3
LimitNOFILE=4096

[Install]
WantedBy=multi-user.target
EOF
```

8. Configure Nginx:
```bash
sudo tee /etc/nginx/sites-available/alerim << EOF
server {
    listen 80;
    server_name your_domain.com;

    root /var/www/alerim;
    index index.html;

    location / {
        try_files \$uri \$uri/ =404;
    }

    location /api {
        proxy_pass http://localhost:8545;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_cache_bypass \$http_upgrade;
    }
}
EOF

sudo ln -s /etc/nginx/sites-available/alerim /etc/nginx/sites-enabled/
sudo rm /etc/nginx/sites-enabled/default
```

9. Deploy web wallet:
```bash
sudo mkdir -p /var/www/alerim
sudo cp -r wallet/web/* /var/www/alerim/
sudo chown -R www-data:www-data /var/www/alerim
```

10. Start services:
```bash
sudo systemctl daemon-reload
sudo systemctl enable alerim-node
sudo systemctl start alerim-node
sudo systemctl restart nginx
```

## SSL Configuration (Optional but Recommended)

Install and configure SSL using Let's Encrypt:

```bash
sudo apt-get install -y certbot python3-certbot-nginx
sudo certbot --nginx -d your_domain.com
```

## Monitoring

Check service status:
```bash
sudo systemctl status alerim-node
sudo systemctl status nginx
```

View logs:
```bash
sudo journalctl -u alerim-node -f
sudo tail -f /var/log/nginx/error.log
```

## Backup

Backup important files:
```bash
sudo cp -r /var/www/alerim /backup/
sudo cp /etc/nginx/sites-available/alerim /backup/
sudo cp /etc/systemd/system/alerim-node.service /backup/
```

## Troubleshooting

1. If the node fails to start:
   - Check logs: `sudo journalctl -u alerim-node -f`
   - Verify permissions: `ls -l /usr/local/bin/alerim-node`
   - Check network connectivity: `netstat -tulpn`

2. If the web wallet is inaccessible:
   - Check Nginx logs: `sudo tail -f /var/log/nginx/error.log`
   - Verify file permissions: `ls -l /var/www/alerim`
   - Test Nginx configuration: `sudo nginx -t`

3. If P2P network issues occur:
   - Check firewall status: `sudo ufw status`
   - Verify port availability: `netstat -tulpn | grep 9000`
   - Test peer connectivity: `telnet peer_address 9000`
