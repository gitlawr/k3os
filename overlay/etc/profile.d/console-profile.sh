#!/bin/bash

# bind <F12>
bind -x '"\e[24~":"harvester-console"'
HARVESTER_DASHBOARD=true TTY=$(tty) harvester-console
