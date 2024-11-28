<<<<<<<, =======, >>>>>>>
<<<<<<< HEAD
Your local changes.
=======
Remote changes from GitHub.
>>>>>>>
# Alerim Cryptocurrency


Alerim (AIM) is a cryptocurrency built on blockchain technology with the following specifications:
- Block time: 1 minute
- Mining rewards: 50 AIM
- Maximum supply: 1,000,000 AIM
- Consensus algorithm: SHA-256
- Merge-mining compatibility enabled

## Features

### Advanced Mining Pool
- Stratum protocol support (v1)
- Dynamic difficulty adjustment (vardiff)
- PPLNS (Pay Per Last N Shares) reward system
- Real-time mining statistics
- Worker-specific difficulty targeting
- Comprehensive performance monitoring

### Mining Pool Specifications
- Default pool fee: 2%
- Payout threshold: 1 AIM
- Maturity depth: 100 confirmations
- Payout interval: 24 hours
- Target share time: 10 seconds
- Vardiff retarget time: 120 seconds

## Project Structure
```
alerim/
├── blockchain/           # Core blockchain implementation
├── cmd/
│   └── alerimnode/      # Mining pool and node implementation
│       ├── mining_pool.go    # Mining pool core
│       ├── stratum.go        # Stratum protocol server
│       ├── vardiff.go        # Variable difficulty system
│       ├── rewards.go        # Reward distribution
│       └── mining_stats.go   # Statistics tracking
├── wallet/              # Wallet implementation
│   ├── web/            # Web-based wallet interface
│   └── cli/            # Command-line wallet tools
├── docs/               # Documentation
└── scripts/            # Deployment and utility scripts
```

## Prerequisites
- Go 1.20+
- Node.js 16+
- Docker (optional)

## Quick Start
1. Clone the repository
2. Install dependencies: `make install`
3. Start the node: `make start-node`
4. Connect to mining pool: `stratum+tcp://localhost:3333`

## Mining Pool Connection
```
URL: stratum+tcp://[host]:3333
Username: [wallet_address]
Password: x
```

## Development Setup
Detailed instructions in [docs/development.md](docs/development.md)

## Deployment
Deployment instructions in [docs/deployment.md](docs/deployment.md)

## Security Considerations
1. Use SSL/TLS in production
2. Keep private keys secure and offline
3. Regularly update system packages
4. Monitor system resources and logs
5. Implement rate limiting for API endpoints
6. Enable DDoS protection
7. Use strong authentication for admin panel

## Performance Monitoring
The mining pool provides comprehensive statistics:
- Real-time hashrate monitoring
- Share submission tracking
- Block finding statistics
- Worker performance metrics
- Difficulty adjustment history
- Payment processing status

## Configuration
Key configuration parameters can be adjusted in config.yaml:
- Block rewards
- Pool fees
- Payout thresholds
- Difficulty adjustment parameters
- Share acceptance policies
- Statistical tracking windows

## License
MIT License - see LICENSE file for details
