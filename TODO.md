 * event configurable
 * strip extension from script basename
 * use first line of script stdout as message
 * fixup readme example for nsca
 * nsca needs a non-empty message
        OK       = echo -e "somehost.example.com;%(event);0;%(message) bla\n" |/usr/sbin/send_nsca -H nagios.example.com -d ";"
