if [[ ${PAM_TYPE} == "open_session" ]]; then
  curl --max-time 10 http://localhost:34000/close
fi
exit 0

# script that closes the tunnel upon a ssh login
# sudo mkdir /etc/pam.scripts
# sudo chmod 0755 /etc/pam.scripts
# sudo cp ssh_login.sh /etc/pam.scripts/ssh_login.sh
# sudo chmod 0700 /etc/pam.scripts/ssh_login.sh
# sudo chown root:root /etc/pam.scripts/ssh_login.sh
# add to end of /etc/pam.d/sshd:
# # SSH Login Script
# session optional pam_exec.so /bin/bash /etc/pam.scripts/ssh_login.sh

# no restart of services required, make sure it works with a new session before logging out with your working one. or you might be sorry