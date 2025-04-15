# Setting up your server
## Creating initial files
Create a folder for your VPN service, put the wsvpn binary in it and set up the default configuration.

Make sure to get the latest binary url.
```bash
mkdir wsvpn
wget  https://github.com/Doridian/wsvpn/releases/download/v5.40.3/wsvpn-linux-amd64 -O wsvpn
chmod +x wsvpn
./wsvpn --print-default-config -mode server > server.yml
```

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