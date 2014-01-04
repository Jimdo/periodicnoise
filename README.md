periodicnoise
===========

NOTE: Work in progress at the moment.

Powerful wrapper for periodic tasks (e.g. controlled by cron), which:
 * scatters the start of it within a random interval, if executed on many machines
 * reports results to your logging system
 * reports state of execution (busy, failure, ok) to your monitoring system
 * skips execution, if a previous execution is still running, reporting this fact
 * ensures that only one task of this name can run at a time
 * terminates running execution, if it takes too long
 * cleans up stale lockfiles of itself (using [github.com/nightlyone/lockfile](https://github.com/nightlyone/lockfile))


[![Build Status][1]][2]

[1]: https://secure.travis-ci.org/Jimdo/periodicnoise.png
[2]: http://travis-ci.org/Jimdo/periodicnoise


LICENSE
-------
BSD

documentation
-------------
[package documentation at go.pkgdoc.org](http://go.pkgdoc.org/github.com/Jimdo/periodicnoise)


quick usage
-----------

TODO

monitoring configuration
------------------------

To enable passive monitoring, you have to configure a command to be executed for
each type of monitoring result. Those commands can be either configured in the
global configuration file `/etc/periodicnoise/config.ini` or in the
user-specific configuration file `$HOME/.config/periodicnoise/config.ini` (the
user-specific settings overwrite the global settings).

Here is a sample configuration:

```
[monitoring]
OK       = printf "somehost.example.com;%(event);0;%(message)\n" |/usr/sbin/send_nsca -H nagios.example.com -d ";"
WARNING  = printf "somehost.example.com;%(event);1;%(message)\n" |/usr/sbin/send_nsca -H nagios.example.com -d ";"
CRITICAL = printf "somehost.example.com;%(event);2;%(message)\n" |/usr/sbin/send_nsca -H nagios.example.com -d ";"
UNKNOWN  = printf "somehost.example.com;%(event);3;%(message)\n" |/usr/sbin/send_nsca -H nagios.example.com -d ";"
```

Note that following strings will be expanded at runtime:

* `%(event)` - monitoring event (name of the executed command)
* `%(state)` - monitoring state, e.g. OK or DEBUG
* `%(message)` - monitoring message

build and install
=================

install from source
-------------------

Install [Go 1][3], either [from source][4] or [with a prepackaged binary][5].

Then run

	go get github.com/Jimdo/periodicnoise
	go get github.com/Jimdo/periodicnoise/cmd/pn

safely execute /bin/true two times

	$GOPATH/bin/pn /bin/true & $GOPATH/bin/pn /bin/true

safely execute /bin/false two times

	$GOPATH/bin/pn /bin/false & $GOPATH/bin/pn /bin/false


[3]: http://golang.org
[4]: http://golang.org/doc/install/source
[5]: http://golang.org/doc/install

LICENSE
-------
BSD

documentation
=============

Instead of writing logging, locking, timeout and load scattering scripts at the 4th company now,
I decided to build an open source tool for it.

problems to be solved
---------------------
 * Scatters the start of it within a random interval, if executed on many machines.
   This ensures central services, we report to, are not overloaded with a spiky load.
   Scattering is by 10% of execution interval by default. So in a 10min execution interval
   the start time for the command to be executed is delayed between 0 and 1 minute.
 * Reports results to your logging system.
   Script results (stderr or stdout and stderr combined) will be collected and reported
   on failure or always to syslog or stderr.
 * Reports state of execution (busy, failure, ok) to your monitoring system.
   Allows simple integration into various monitoring systems. Never miss failing crons anymore.
 * skips execution, if a previous execution is still running, reporting this fact
   Avoid unwanted parallel execution of scripts that get too slow over time or only sometimes.
   But still report this slowless via monitoring, so the devops/sysadmin can take atcion, if desired.
 * Retry logic and report only final failure.
   Retry known flaky commands, which failed to execute. Allows to increase robustness and avoid flapping monitorings.
 * Ensures that only one task of this name can run at a time.
   Avoid unwanted parallel execution of scripts that do not support it or lead to high load, if running multiple times (e.g. backups)
 * Terminates running execution, if it takes too long.
   Allows to have strict timing control for nice to have cron jobs or cron jobs, that violate timing contraints sometimes.
 * cleans up stale lockfiles of itself (using [github.com/nightlyone/lockfile](https://github.com/nightlyone/lockfile))
   Heal itself after `kill -9` or power outages on badly configured systems.

contributing
============

Contributions are welcome. Please open an issue or send me a pull request for a dedicated branch.
Make sure the git commit hooks show it works.

git commit hooks
-----------------------
enable commit hooks via

        cd .git ; rm -rf hooks; ln -s ../git-hooks hooks ; cd ..



[![Bitdeli Badge](https://d2weczhvl823v0.cloudfront.net/Jimdo/periodicnoise/trend.png)](https://bitdeli.com/free "Bitdeli Badge")

