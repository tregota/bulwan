# bulwan
Docker Container that connects to a server using SSH and redirects a port on it to local http services on you LAN like a REST API.

My first Go project so use it at your own peril.. :)

I have a personal webpage on a VPS where I want to use databases and other resources that I have on my LAN, but I don't want to open ports in my router and expose them to the Internet. This allows me to feel a bit safer while not having to pay for cloud storage. ;)

bulwan is a play on the Swedish word bulvan which is a legal term for a person that acts in your stead so that you may remain anonymous.

## Environmental Variables
Since I'm using it in a docker container, all settings are fetched from these environmental variables:

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
* HTTP_GET_ON_CLOSE
  - An URL that gets called if the close tunnel command is called. Couldn't get it to work with https unless I use InsecureSkipVerify, so not perfect.. I use it to send notifications to my phone.
* EXPOSED_HTTPSERVER_PREFIX_1
  - The first URL prefix. e.g. proxy to redirect calls to /proxy/{etc} to the URL given in EXPOSED_HTTPSERVER_URL_1 with {etc} appended.
* EXPOSED_HTTPSERVER_URL_1
  - the URL to which calls using the EXPOSED_HTTPSERVER_PREFIX_1 route prefix are redirected to. Don't end with a slash.
* EXPOSED_HTTPSERVER_PREFIX_2
  - another prefix. You can add how many redirects you'd like as long as your don't skip over any indexes.
* EXPOSED_HTTPSERVER_URL_2
  - another URL. You can add how many redirects you'd like as long as your don't skip over any indexes.

## Commands
The router handles commands in addition to the redirects, these are:

* /close
  - closes the SSH connection and prevents reconnects until /open has been called. Can be called from the server if someone connects which, hopefully, make breaches harder.
* /open
  - allows it to connect, must be called after deployment since the default is not allowing a connection.

bulwan binds both the configured port on the remote server and :35300 locally.
