Identify Mikrotik PPPOE-index.
From Winbox , go to IP -> SNMP 

->Enable
->Community -> New -> Add community string (e.g. community_string) -> OK

From a remote terminal , do : snmpwalk -v 2c -c <community_string> <ip_mikrotik> 1.3.6.1.2.1.2.2.1.2 | grep -i "pppoe"

The return will be something like this :
iso.3.6.1.2.1.2.2.1.2.29 = STRING: "pppoe-out1"
Then, we can use the index 29 to get the speed.

--------------------------------------------------------

Identify Singbox API.
If use the config-singbox-sample.json, the API is http://[IP_ADDRESS_OF_SINGBOX_MACHINE]:9090

--------------------------------------------------------

All other required tokens and stuffs, ask googlel/AI to show steps to get it.



