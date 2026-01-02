# SPDX-FileCopyrightText: 2026 Comcast Cable Communications Management, LLC
# SPDX-License-Identifier: Apache-2.0
#!/usr/bin/env sh
set -e

# check arguments for an option that would cause /xmidt-agent to stop
# return true if there is one
_want_help() {
    local arg
    for arg; do
        case "$arg" in
            -'?'|-h|--help|-s|--show)
                return 0
                ;;
        esac
    done
    return 1
}

_main() {
    # if command starts with an option, prepend wrp-kafka-splitter
    if [ "${1:0:1}" = '-' ]; then
        set -- /wrp-kafka-splitter "$@"
    fi

    # skip setup if they aren't running /wrp-kafka-splitter or want an option that stops /wrp-kafka-splitter
    if [ "$1" = '/wrp-kafka-splitter' ] && ! _want_help "$@"; then
        echo "Entrypoint script for wrp-kafka-splitter Client ${VERSION} started."
    fi

    exec "$@"
}

_main "$@"