# Setting up your server
NOTE: I am using a linux host with systemd here.

## Creating initial files
Create a folder for your VPN service, put the wsvpn binary in it and set up the default configuration.
Make sure to get the latest binary url.

```bash
mkdir wsvpn
wget  https://github.com/Doridian/wsvpn/releases/download/v5.40.3/wsvpn-linux-amd64 -O wsvpn
chmod +x wsvpn
./wsvpn --print-default-config -mode server > server.yml
```

## Configuring your server
First, configure an interface name for your VPN.
I recommend you to leave it as `persist` in your system, as you might want to apply advanced firewall configurations in the future.
```yaml
name: "vpn0"
persist: true
```

Then, configure your VPN's pool of addreses.
```yaml
subnet: 192.168.3.0/24
```
Notice that your VPN host will always take the first IP on this network. This will become your `VPN's gateway`.
In this example, it would take 192.168.3.1.

Now you need to configure your `listen ipv4 address`. This will either prevent external access without a middleman, or just allow anyone to connect to it using your IP:port.

To allow everyone to access your VPN:
```yaml
server:
  listen: 0.0.0.0:9000
```

To allow only an app in your host to access your VPN:
```yaml
server:
  listen: 127.0.0.1:9000
```

Finally, you might want to setup some real security on your VPN.
I recommend reading `Example: VPN with TLS and htpasswd authentication` to allow access only via a password, or `Example:-VPN-with-TLS-and-mTLS` to allow access only with a client SSL certificate.

By default, your VPN will allow just anyone to access it.

For further security enhancements, consider using a reverse proxy (I talk more about it at the end of this doc).

## Setting up a systemd service
To allow your system to automatically start and stop your VPN, you must create a system service.

Here's a linux systemd service file for all of this:

```bash
cat >> /etc/systemd/system/wsvpn.service << EOF
[Unit]
Description=WebSocket VPN server
After=network.target

[Service]
WorkingDirectory=/your-full-path-here/wsvpn
Type=simple
Restart=always
ExecStart=/your-full-path-here/wsvpn/wsvpn -mode "server" -config "server.yml"

[Install]
WantedBy=multi-user.target
EOF
```

Also, you usually want to allow all VPN users to access the network via NAT (they interact with it via your system's IP). This can be accomplished by adding a few more iptables lines on your service:

```bash
cat >> /etc/systemd/system/wsvpn.service << EOF
[Unit]
Description=WebSocket VPN server
After=network.target

[Service]
WorkingDirectory=/your-full-path-here/wsvpn
Type=simple
Restart=always
ExecStart=/your-full-path-here/wsvpn/wsvpn -mode "server" -config "server.yml"

# To allow users to interact with your network, you need to configure your system to work as a router with NAT
# Replace 192.168.3.0/24 with your desired VPN network
# Replace ens18 with your real interface name

ExecStartPost=echo 1 > /proc/sys/net/ipv4/ip_forward
ExecStartPost=iptables -t nat -A POSTROUTING -s 192.168.3.0/24 -o ens18 -j MASQUERADE
ExecStop=iptables -t nat -D POSTROUTING -s 192.168.3.0/24 -o ens18 -j MASQUERADE

# In the rare case that your system uses DOCKER, this is needed, as docker by default doesnt allow your system to work as a router
ExecStartPost=iptables -P FORWARD ACCEPT
ExecStop=iptables -P FORWARD DROP


[Install]
WantedBy=multi-user.target
EOF
```

## Reverse proxy configuration
In a real world scenario the best setup is to have a middleman handle connections for you. 

Since wsvpn uses websockets, that is called a *reverse proxy*.

Here I will show you an example working config for the webserver Apache2:

### Apache2 reverse proxy entry (wsvpn.conf)
```apache
<VirtualHost *:443>
	# SSL config
    Include cert.conf
	
    ServerName wsvpn.mydomain.com
	ServerAlias www.wsvpn.mydomain.com

	# Websocket support
	RewriteEngine On
	RewriteCond %{HTTP:Connection} Upgrade [NC]
	RewriteCond %{HTTP:Upgrade} websocket [NC]
	RewriteRule /(.*) ws://localhost:9000/$1 [P,L]
	# Proxy
	ProxyPreserveHost on
	ProxyPass /myprivateid/ http://localhost:9000/
	ProxyPassReverse /myprivateid/ http://localhost:9000/
	SetEnvIf X-Forwarded-Proto "https" HTTPS=on
	RequestHeader set X-Forwarded-Proto "https"
	RequestHeader set X-Forwarded-Port "443"
	ErrorLog ${APACHE_LOG_DIR}/wsvpn_error.log
	CustomLog ${APACHE_LOG_DIR}/wsvpn_access.log combined
</VirtualHost>
<VirtualHost *:80>
	ServerName wsvpn.mydomain.com
	ServerAlias www.wsvpn.mydomain.com
	RedirectPermanent / https://wsvpn.mydomain.com/
</VirtualHost>
```
If you noticed I used **/myprivateid/** as a location in my webpage. This is called obfuscation.

For security purposes, never expose your VPN on the root of your webpage, nor use something as common as /VPN/. Bots will find it and exploit it.

In this example, clients would access my VPN via 
wss://wsvpn.mydomain.com/myprivateid/

Other ways to protect your VPN from unauthorized access are:
- Use a fake domain name configured by the client.
- Require certain IPs using the "require ip" statement.
- Set up required custom http headers in both server.yml and client.yml
- Set up htpasswd authentication.
- Set up mutual TLS so clients require an SSL client cert to enter. 
- Change the default listening port of your webserver to a random one