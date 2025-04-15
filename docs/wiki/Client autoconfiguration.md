# Client autoconfiguration
In most other VPN solutions out there you usually have:

- A custom routing table, to allow users to connect to different networks, without rerouting all traffic through the VPN (saving precious latency & bandwith).
- A custom DNS server on connection, to allow users access to private webpages and services on the network.

This describes how to set up your client files to auto configure  IP routing and DNS on VPN connection.

# Windows
### Initial file structure

Download the latest windows binary from https://github.com/Doridian/wsvpn/releases/ , then execute it to create a default client configuration file

./wsvpn-windows-amd64.exe --print-default-config -mode client > client.yml

After this, configure this `client.yml` file to connect to your server URL with your preffered auth method.

Create a batch file, here called `connect.bat`, with the following:

#### connect.bat
```batch
cd C:\wsvpn
wsvpn-windows-amd64.exe -mode "client" -config "client.yml"
```
Then, execute it as administrator to stablish a fresh connection. 
This should create a new `wintun.dll` file and a windows adapter that will dissapear on disconection, called `WaterWinTunInterface`.

Do not close this connection, as we will need this adapter to appear on windows.

You should end up with this directory structure:
```
├── wsvpnclient-windows
    ├── wsvpn-windows-amd64.exe
    ├── wintun.dll
    ├── connect.bat
    └── client.yml
```

### Creating a powershell script
In order to configure routing & dns in windows, you must use powershell commands, ran as administrator.

First, you need to get the ID windows gave to your new `WaterWinTunInterface` adapter.

```powershell
(Get-NetAdapter | Where-Object { $_.Name -eq "WaterWinTunInterface" }).InterfaceIndex
```

Then you can configure your routing table.

In windows, your routing table can be seen by doing:
```powershell
route print
```

**Example:** your client wants access to your main LAN. Its network address is `192.168.1.0/24`.

So you need to add an entry in his routing table, telling windows to ask all related to this `192.168.1.0/24` network to the VPN interface.

```powershell
route add 192.168.1.0 mask 255.255.255.0 192.168.3.1 IF $interfaceIndex
```
Notice I gave the IP `192.168.3.1` before the interface ID, this is the VPN's server IP, or as commonly known, the VPN's `gateway`. You can ask this IP for any other IP addresses.

To automate all this process at the end, you will end up with this script:
#### client.ps1
```powershell
$interfaceIndex = (Get-NetAdapter | Where-Object { $_.Name -eq "WaterWinTunInterface" }).InterfaceIndex

route add 192.168.1.0 mask 255.255.255.0 192.168.3.1 IF $interfaceIndex
```

Now, what if your client wants to access all private domains in your network, for example, login.private.com, that is only known by `192.168.1.250`?

You must tell windows to use your private DNS IP. In this example, `192.168.1.250`. It is as simple as adding this to your script:

#### client.ps1
```powershell
$interfaceIndex = (Get-NetAdapter | Where-Object { $_.Name -eq "WaterWinTunInterface" }).InterfaceIndex

route add 192.168.1.0 mask 255.255.255.0 192.168.3.1 IF $interfaceIndex

Set-DnsClientServerAddress -InterfaceIndex $interfaceIndex -ServerAddresses 192.168.1.250
```

### Telling WSVPN to execute client.ps1
Simply find these lines and edit them as follows in your `client.yml` file:

```yaml
scripts:
  # These scripts get run as "args... operation subnet interface"
  # Pass in an array, first argument is the executable, further arguments
  # are used before WSVPN provided arguments
  # Example: "./handler.sh up 192.168.3.2/24 tun0"
  up: ["powershell.exe", "-ExecutionPolicy", "Bypass", "-File", "client.ps1"]
  down: []
```

All done! everytime you run `connect.bat` as administrator, you will get access to your main LAN, and also your private DNS domains.

## Final windows client directory structure
```
├── wsvpnclient-windows
    ├── wsvpn-windows-amd64.exe
    ├── wintun.dll
    ├── connect.bat
    ├── client.yml
    └── client.ps1
```

# Linux
It is way, way easier to acomplish a custom routing table & DNS on linux.

### Initial file structure

Download the latest linux binary from https://github.com/Doridian/wsvpn/releases/ , then execute it to create a default client configuration file

./wsvpn-linux-amd64
 --print-default-config -mode client > client.yml

After this, configure this `client.yml` file to connect to your server URL with your preffered auth method.

Create a bash file, here called `connect.sh`, with the following:

#### connect.sh
```batch
wsvpn-linux-amd64 -mode "client" -config "client.yml"
```
Then, execute it as `root` to stablish a fresh connection. 
This should create a new linux tap interface that will dissapear on disconection, called `vpn0` or whatever you configured in your `client.yml` file.

Do not close this connection, as we will need this adapter to appear on linux.

You should end up with this directory structure:
```
├── wsvpnclient-linux
    ├── wsvpn-linux-amd64
    ├── connect.sh
    └── client.yml
```
### Creating a bash setup script
Almost all linux distros come with the ip software stack.

All you need to know is your interface name, in this case, `vpn0`.

In linux, your routing table can be seen by doing:
```powershell
ip route
```

To configure your routing table to forward all `192.168.1.0/24` (aka your main LAN), you just do:

```bash
ip route add 192.168.1.0/24 via 192.168.3.1 dev vpn0
```

Then, to set your private DNS you must add it as the first entry in your `/etc/resolv.conf`:
```bash
sed -i '1i nameserver 192.168.1.250' /etc/resolv.conf
```

Your final up script would look like:

#### client.sh
```bash
ip route add 192.168.1.0/24 via 192.168.3.1 dev vpn0
sed -i '1i nameserver 192.168.1.250' /etc/resolv.conf
```


Keep in mind that all these changes will stay even on disconnection. That is just the way linux works.

For this configuration to be removed, you must create another script that deletes it on disconnection:

#### clientdown.sh
```bash
ip route del 192.168.1.0/24 via 192.168.3.1 dev vpn0
sed -i '/^nameserver 192.168.1.250$/d' /etc/resolv.conf
```

### Telling WSVPN to execute client.sh & clientdown.sh
Simply find these lines and edit them as follows in your `client.yml` file:

```yaml
scripts:
  # These scripts get run as "args... operation subnet interface"
  # Pass in an array, first argument is the executable, further arguments
  # are used before WSVPN provided arguments
  # Example: "./handler.sh up 192.168.3.2/24 tun0"
  up: ["bash", "client.sh"]
  down: ["bash", "clientdown.sh"]
```

## Final linux client directory structure
```
├── wsvpnclient-linux
    ├── wsvpn-linux-amd64
    ├── connect.sh
    ├── client.yml
    ├── client.sh
    └── clientdown.sh
```