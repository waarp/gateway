#!/usr/bin/env bash

ACTION=$1
shift

#####################################################################
###  TASKS
#####################################################################

t_test() {
    go test $@ ./cmd/... ./pkg/...
}

t_check() {
    local CI_VERSION=$(grep GOLANGCI_LINT_VERSION .gitlab-ci.yml | head -n 1 | cut -d' ' -f4)
    if ! $(which golangci-lint >/dev/null 2>&1); then
        echo "golangci-lint cannot be found. Please, install it and re-run checks"
        return 1
    fi
    if ! $(golangci-lint --version | grep $CI_VERSION >/dev/null 2>&1); then
        echo "***********************************************"
        echo "WARNING"
        echo "***********************************************"
        echo "Your version os golangci-lint is: $(golangci-lint --version | sed 's:.*\s\([0-9\.]\+\)\s.*:\1:')"
        echo "CI runs version $CI_VERSION"
        echo "***********************************************"
    fi
    go vet ./cmd/... ./pkg/...
    golangci-lint run \
        --enable-all --disable goconst,depguard,gochecknoglobals,gochecknoinits,gocritic,interfacer,maligned,prealloc,lll,wsl \
        --max-issues-per-linter 0 --max-same-issues 0 \
        --exclude-use-default=false  \
        --exclude 'Potential file inclusion via variable' \
        --exclude 'Error return value of .((os\.)?std(out|err)\..*|.*Close|.*Flush|os\.Remove(All)?|.*printf?|os\.(Un)?Setenv). is not checked' \
        --exclude 'unnecessary leading newline' \
        --exclude 'unnecessary trailing newline' \
        --exclude 'ST1000' \
        --exclude 'SA5008' \
        --exclude 'Line contains TODO/BUG/FIXME' \
        --skip-dirs .gocache \
    	  --tests=false
}

t_test_watch() {
    goconvey -launchBrowser=false -port=8081 -excludedDirs=doc $@
}

t_doc() {
    cd doc
    .venv/bin/sphinx-build source/ build/html/
}

t_doc_watch() {
    cd doc
    .venv/bin/sphinx-autobuild -p 8082 source/ build/html/
}

t_usage() {
    echo "Usage $0 [ACTION]"
    echo ""
    echo "Available actions"
    echo ""
    echo "  check       Run static analysis"
    echo "  test        Run tests"
    echo "  test watch  Starts convey to watch code and run tests when"
    echo "              it has been changed"
    echo "  doc         Builds the doc"
    echo "  doc watch   Watch the source of the documentation and builds it when"
    echo "              it has been changed"
    echo ""
}

#####################################################################
###  MAIN
#####################################################################

case $ACTION in
    check)
        t_check
        ;;

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

    doc)
        SUB=$1
        case $SUB in
            watch)
                shift
                t_doc_watch $@
                ;;
            *)
                t_doc $@
                ;;
        esac
        ;;

    *)
        t_usage
        ;;
esac
