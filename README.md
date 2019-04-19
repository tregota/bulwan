# bulwan
Application that connects to a server using SSH and redirects a port on it to local http services on your LAN.
My first Go project so use it at your own peril.. :)

Can be built into a docker container.

I have a personal webpage on a VPS where I want to use databases and other resources that I have on my LAN, but I don't want to open ports in my router and expose them to the Internet. This allows me to feel a bit safer while not having to pay for cloud storage. ;)

bulwan is a slight play on the Swedish word bulvan which is a legal term for a person that acts in your stead so that you may remain anonymous.

## Environmental Variables
Since it can be run in a docker container all settings can be fetched from these environmental variables that override whatever is in settings.conf

* SERVER_HOST
  - The host to connect to.
* SERVER_PORT
  - The SSH port on the host.
* SERVER_PUBLIC_KEY
  - The public key, I copied it from known_hosts. Specifically the longest string of characters, after the key type.
* SERVER_PUBLIC_KEY_TYPE
  - The type of the public key, also from the known_hosts file. Something like ssh-rsa or ecdsa-sha2-nistp256.
* SSH_USERNAME
  - The user you have created on the server. Should be used only for this (since if it cannot bind the port it will try to kill all other connections by the same user) and probably have restricted access.
* SSH_LISTEN_PORT
  - The port to bind on the server for the reverse tunnel.
* SSH_PRIVATE_KEY
  - This is the content of your PEM formatted private key file. the part declared using -----BEGIN RSA PRIVATE KEY----- and -----END RSA PRIVATE KEY-----.
* DIAL_CLOSE_DELAY
  - If you use a PAM ssh login script on the server to kill the tunnel then it is a good idea to set this delay in seconds after a dial that it will not allow closing of the tunnel. So it doesn't close itself (usually not a problem since the tunnel is closed while dialing but I have seen it happen.)
* HTTP_GET_ON_CLOSE
  - An URL that gets called if the close tunnel command is called. I use it to send notifications to my phone. The docker container doesn't have any root certificates available so there it's better to call local services that in turn can then make more secure calls to the Internet if that is what you need.
* EXPOSED_HTTP_SERVERS_PREFIX_1
  - The first URL prefix. e.g. proxy to redirect calls to /proxy/{etc} to the URL given in EXPOSED_HTTP_SERVERS_URL_1 with {etc} appended.
* EXPOSED_HTTP_SERVERS_URL_1
  - The URL to which calls using the EXPOSED_HTTP_SERVERS_PREFIX_1 route prefix are redirected to. Don't end with a slash.
* EXPOSED_HTTP_SERVERS_PREFIX_2
  - Another prefix. You can add how many redirects you'd like as long as your don't skip over any indexes.
* EXPOSED_HTTP_SERVERS_URL_2
  - Another URL. You can add how many redirects you'd like as long as your don't skip over any indexes.

## Other configuration options
You can add files named for a setting name from the settings.conf to for instance have the private key in a file called SSHPrivateKey.

## Commands
The router handles commands in addition to the redirects, these are:

* /close
  - closes the SSH connection and prevents reconnects until /open has been called. Can be called from the server if someone connects which, hopefully, make breaches harder.
* /open
  - allows it to connect, must be called after deployment since the default is not allowing a connection.

bulwan binds both the configured port on the remote server and :35300 (default) locally.
