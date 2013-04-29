 * use first line of script stdout as message
 * fixup readme example for nsca
        OK       = echo -e "somehost.example.com;%(event);0;%(message) bla\n" |/usr/sbin/send_nsca -H nagios.example.com -d ";"
