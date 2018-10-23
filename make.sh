#!/usr/bin/env bash

ACTION=$1
shift

#####################################################################
###  TASKS
#####################################################################

t_test() {
    go test $@ ./cmd/... ./pkg/...
}

t_test_watch() {
    goconvey -launchBrowser=false -port=8081 $@
}

t_usage() {
    echo "Usage $0 [ACTION]"
    echo ""
    echo "Available actions"
    echo ""
    echo "  test        Run tests"
    echo "  test watch  Starts convey to watch code and run tests when"
    echo "              it has been changed"
    echo ""
}

#####################################################################
###  MAIN
#####################################################################

case $ACTION in
    test)
        SUB=$1
        case $SUB in
            watch)
                shift
                t_test_watch $@
                ;;
            *)
                t_test $@
                ;;
        esac
        ;;

    *)
        t_usage
        ;;
esac
