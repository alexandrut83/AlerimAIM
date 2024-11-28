# Alerim Mining Server Installation Guide

## Server Specifications
- Server Address: http://147.78.130.55/
- Repository: https://github.com/alexandrut83/alerimAIM
- Branch: main

## Prerequisites
1. Ubuntu Server 20.04 LTS or higher
2. Minimum Requirements:
   - 4 CPU cores
   - 8GB RAM
   - 100GB SSD
   - Static IP address

## Installation Steps

### 1. System Preparation
```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install required packages
sudo apt install -y build-essential git curl wget nginx postgresql redis-server
```

### 2. Install Go
```bash
wget https://go.dev/dl/go1.20.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.20.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

### 3. Clone Repository
```bash
git clone https://github.com/alexandrut83/alerimAIM.git
cd alerimAIM/CascadeProjects/alerim2
```

### 4. Database Setup
```bash
sudo -u postgres psql
CREATE DATABASE alerim_pool;
CREATE USER alerim_user WITH ENCRYPTED PASSWORD '153118!Alex';
GRANT ALL PRIVILEGES ON DATABASE alerim_pool TO alerim_user;
```

### 5. Configuration
Create config.yaml in the config directory with the following settings:
```yaml
server:
  host: "147.78.130.55"
  stratum_port: 3333
  api_port: 8080
  admin_port: 8081

admin:
  username: "alex11alerim"
  password: "153118!Alex"

mining:
  network: "mainnet"
  block_reward: 50
  pool_fee: 2.0
```

### 6. Build and Install
```bash
make clean
make install
make build
```

### 7. Setup Systemd Service
Create `/etc/systemd/system/alerim.service`:
```ini
[Unit]
Description=Alerim Mining Pool
After=network.target postgresql.service

[Service]
User=alerim
Group=alerim
WorkingDirectory=/opt/alerim
ExecStart=/usr/local/bin/alerimnode
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### 8. Start Services
```bash
sudo systemctl daemon-reload
sudo systemctl enable alerim
sudo systemctl start alerim
```

### 9. Setup Nginx Reverse Proxy
Create `/etc/nginx/sites-available/alerim`:
```nginx
server {
    listen 80;
    server_name 147.78.130.55;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }

    location /admin {
        proxy_pass http://localhost:8081;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
```

### 10. Enable and Start Nginx
```bash
sudo ln -s /etc/nginx/sites-available/alerim /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

## Testing the Installation

### 1. Check Services
```bash
sudo systemctl status alerim
sudo systemctl status nginx
sudo systemctl status postgresql
```

### 2. Test Mining Connection
Use a mining client to connect:
```
URL: stratum+tcp://147.78.130.55:3333
Username: your_wallet_address
Password: x
```

### 3. Access Admin Panel
Visit `http://147.78.130.55/admin`
- Username: alex11alerim
- Password: 153118!Alex

## Monitoring and Logs

### View Logs
```bash
# Mining pool logs
sudo journalctl -u alerim -f

# Nginx access logs
sudo tail -f /var/log/nginx/access.log

# Nginx error logs
sudo tail -f /var/log/nginx/error.log
```

## Security Recommendations

1. Enable UFW Firewall:
```bash
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 3333/tcp
sudo ufw enable
```

2. Setup SSL/TLS:
```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d your_domain.com
```

3. Change default passwords
4. Enable fail2ban
5. Regular system updates
6. Monitor system resources

## Troubleshooting

1. Check logs for errors:
```bash
sudo journalctl -u alerim -f
```

2. Verify ports are open:
```bash
sudo netstat -tulpn | grep LISTEN
```

3. Test database connection:
```bash
psql -h localhost -U alerim_user -d alerim_pool
```

4. Check service status:
```bash
sudo systemctl status alerim
```

## Support and Updates

For support issues:
1. Check logs for errors
2. Visit GitHub repository issues page
3. Contact support team

Regular updates:
```bash
cd /path/to/alerim
git pull
make build
sudo systemctl restart alerim
```
