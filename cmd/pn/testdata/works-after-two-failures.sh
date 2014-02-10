#!/bin/sh
set -e

if [ "x$1" = "x" ]; then
    printf "usage: %s statefile\n" $(basename -- $0)
    exit 254
elif ! [ -w "$1" ]; then
    printf "statefile %s must be writable\n" "$1"
    exit 253
fi

statefile="$1"
state="$(cat "$statefile")"

case "x$state" in
    "x"|"x0")
        printf "1" > $statefile
        exit 1
        ;;
    "x1")
        printf "2" > $statefile
        exit 1
        ;;
    "x2")
        printf "3" > $statefile
        exit 0
        ;;
    *)
        printf "statefile %s corrupted, content is %s\n" "$statefile" "$state"
        exit 252
        ;;
esac
