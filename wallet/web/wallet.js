class AlerimWallet {
    constructor() {
        this.web3 = new Web3(new Web3.providers.HttpProvider('http://localhost:8545'));
        this.initializeEventListeners();
    }

    initializeEventListeners() {
        document.getElementById('generateWallet').addEventListener('click', () => this.generateWallet());
        document.getElementById('importWallet').addEventListener('click', () => this.importWallet());
        document.getElementById('sendTransaction').addEventListener('click', () => this.sendTransaction());
    }

    async generateWallet() {
        try {
            const account = this.web3.eth.accounts.create();
            this.displayWalletInfo(account);
            await this.updateBalance(account.address);
        } catch (error) {
            console.error('Error generating wallet:', error);
            alert('Failed to generate wallet. Please try again.');
        }
    }

    async importWallet() {
        const privateKey = prompt('Enter your private key:');
        if (!privateKey) return;

        try {
            const account = this.web3.eth.accounts.privateKeyToAccount(privateKey);
            this.displayWalletInfo(account);
            await this.updateBalance(account.address);
        } catch (error) {
            console.error('Error importing wallet:', error);
            alert('Invalid private key. Please try again.');
        }
    }

    displayWalletInfo(account) {
        const walletInfo = document.getElementById('walletInfo');
        const publicAddress = document.getElementById('publicAddress');
        const privateKey = document.getElementById('privateKey');

        walletInfo.classList.remove('d-none');
        publicAddress.value = account.address;
        privateKey.value = account.privateKey;
    }

    async updateBalance(address) {
        try {
            const balance = await this.web3.eth.getBalance(address);
            const balanceInAim = this.web3.utils.fromWei(balance, 'ether');
            document.getElementById('balance').textContent = balanceInAim;
        } catch (error) {
            console.error('Error updating balance:', error);
        }
    }

    async sendTransaction() {
        const recipientAddress = document.getElementById('recipientAddress').value;
        const amount = document.getElementById('amount').value;
        const privateKey = document.getElementById('privateKey').value;

        if (!recipientAddress || !amount || !privateKey) {
            alert('Please fill in all fields');
            return;
        }

        try {
            const account = this.web3.eth.accounts.privateKeyToAccount(privateKey);
            const nonce = await this.web3.eth.getTransactionCount(account.address);
            const gasPrice = await this.web3.eth.getGasPrice();
            
            const tx = {
                from: account.address,
                to: recipientAddress,
                value: this.web3.utils.toWei(amount.toString(), 'ether'),
                gas: '21000',
                gasPrice: gasPrice,
                nonce: nonce
            };

            const signedTx = await this.web3.eth.accounts.signTransaction(tx, privateKey);
            const receipt = await this.web3.eth.sendSignedTransaction(signedTx.rawTransaction);
            
            alert('Transaction sent successfully!');
            await this.updateBalance(account.address);
        } catch (error) {
            console.error('Error sending transaction:', error);
            alert('Failed to send transaction. Please try again.');
        }
    }
}

// Initialize wallet when page loads
window.addEventListener('load', () => {
    window.alerimWallet = new AlerimWallet();
});

// Utility functions
function copyToClipboard(elementId) {
    const element = document.getElementById(elementId);
    element.select();
    document.execCommand('copy');
    alert('Copied to clipboard!');
}

function togglePrivateKey() {
    const privateKeyInput = document.getElementById('privateKey');
    privateKeyInput.type = privateKeyInput.type === 'password' ? 'text' : 'password';
}
