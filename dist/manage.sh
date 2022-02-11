#!/bin/bash

CURDIR=$(cd "$(dirname "$0")" && pwd)

PIDFILE=$CURDIR/../log/gatewayd.pid
DAEMONNAME=waarp-gatewayd
DAEMONPATH=$CURDIR/../bin/$DAEMONNAME
STARTLOG=$CURDIR/../log/startup.log
DAEMON_PARAMS="server -c $CURDIR/../etc/gatewayd.ini"

export PATH=$CURDIR/../share:$CURDIR/../bin:$PATH


cd "$CURDIR/.." || exit 2

ACTION=$1
shift;

do_start() {
    local rv
    if do_status >/dev/null ; then
        echo "Waarp Gateway is already running."
        rv=2
    else 
        nohup "$DAEMONPATH" $DAEMON_PARAMS >> "$STARTLOG" 2>&1 &
        local pid=$!
        sleep 2
        if pgrep -f $DAEMONNAME | grep $pid >/dev/null 2>&1; then
            echo "Waarp Gateaway is started"
            echo $pid > "$PIDFILE"
            rv=0
        else
            echo "Waarp Gateway could not start. See $STARTLOG for more information."
            rv=1
        fi
    fi
    return $rv
}

do_stop() {
    local rv
    if do_status >/dev/null ; then
        echo -n "Waarp Gateway has been asked to exit. It May take some time"
        if kill -SIGTERM "$(cat "$PIDFILE")"; then
            for i in $(seq 1 24); do
                if pgrep -f $DAEMONNAME | grep "$(cat "$PIDFILE")" >/dev/null 2>&1; then
                    echo -n "."
                else
                    echo
                    echo "Waarp Gateway has been stopped."
                    rm "$PIDFILE"
                    rv=0
                    return $rv
                fi
                sleep 5
            done

            echo
            echo "Waarp Gateway is still waiting for backround jobs to stop before exiting."
            echo "It will exit once all subprocesses are done."
            rv=3
        else
            echo "Waarp Gateway could not be stopped."
            rv=1
        fi
    else
        echo "Server is not running"
        rv=2
    fi
    return $rv
}

do_force_stop() {
    local rv
    if do_status >/dev/null ; then
        if kill -SIGKILL "$(cat "$PIDFILE")"; then
            echo "Waarp Gateway has been killed."
            rm "$PIDFILE"
            rv=0
        else
            echo "Waarp Gateway could not be killed."
            rv=1
        fi
    else
        echo "Server is not running"
        rv=2
    fi
    return $rv
}

do_status() {
    local rv
    if [[ -f $PIDFILE ]]; then
        kill -0 "$(cat "$PIDFILE")" >/dev/null 2>&1
        rv=$?
    else
        rv=2
    fi
    case $rv in
        0)
            echo "Running."
            ;;
        1)
            echo "Not running."
            ;;
        2)
            echo "No PID file found."
            ;;
        *)
            echo "Unknown status."
            ;;
    esac
    return $rv
}

do_restart() {
    local rv
    do_stop
    rv=$?
    if [[ $rv == 0 ]] ; then
        do_start
        rv=$?
    fi
    return $rv
}

case $ACTION in
    start)
        do_start "$*"
        ;;
    stop)
        do_stop
        ;;
    force-stop)
        do_force_stop
        ;;
    status)
        do_status
        ;;
    restart)
        do_restart
        ;;
    *)
        echo "Use one of these commands: status, start, stop, force-stop, restart"
        ;;
esac
