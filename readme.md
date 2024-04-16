
# WebSocket Multiplexing Proxy

## Without TLS _(It's not safe)_:

### Run Server
```bash
./simpleproxy -server -bind :8080 -token mypass1234 
```

### Run client
```bash
./simpleproxy -client -bind :1080 -server-address ws://127.0.0.1:8080 -token mypass1234
```

## With TLS

You have to get an **SSL/TLS certificate**, you can use **Certbot**; Certbot is a free, open source software tool for automatically using Let's Encrypt certificates on manually-administrated websites to enable HTTPS

On a linux distribute run this commands:

```bash
sudo apt install certbot
sudo certbot certonly --standalone --agree-tos -d [YOUR DOMAIN]
```

**Certbot** give you public-key (`fullchain.pem`) and private-key (`privkey.pem`), now use them to run **simple proxy** with SSL/TLS support.

### Run Server
```bash
./simpleproxy -server -bind :8080 -token mypass1234 -public-key /etc/letsencrypt/live/[YOUR DOMAIN]/fullchain.pem -private-key /etc/letsencrypt/live/[YOUR DOMAIN]/privkey.pem 
```

### Run client
```bash
./simpleproxy -client -bind :1080 -server-address wss://[YOUR DOMAIN]:8080 -token mypass1234
```

On local system, **Simple Proxy** run's socks4, socks5 and http proxy on port 1080
