#!/bin/sh

#  ASCII Roulette v1.0.0
#
#  presented by:
#
#     ||`              '||`
#     ||   ''           ||
# .|''||   ||   '''|.   ||  '||  ||` '||''|,    .|'', .|''|, '||),,(|,
# ||  ||   ||  .|''||   ||   ||  ||   ||  ||    ||    ||  ||  || || ||
# `|..||. .||. `|..||. .||.  `|..'|.  ||..|' .. `|..' `|..|' .||    ||.
#                                     ||
#  (we're hiring!)                   .||
#
#
# ASCII Roulette is a command-line based video chat client that connects
# you with a random person. It was written as a demonstration of the
# Pion WebRTC library.
#
# [[ This project is completely open source! ]]
# Github: https://github.com/dialupdotcom/ascii_roulette
# Website: https://dialup.com/ascii
# Email: webmaster@dialup.com

function fail_unsupported() {
    platform="${1:-"your platform"}"
    echo "ASCII Roulette isn't supported on $platform yet :-( Try Mac or Linux instead."
    echo ""
    echo "Contributions are welcome! https://github.com/dialupdotcom/ascii_roulette"
    exit 1
}

function detect_platform() {
    case $(uname | tr '[:upper:]' '[:lower:]') in
        linux*)
            case $(uname -m) in
                x86_64)
                    echo "linux64"
                ;;
                *)
                    fail_unsupported "32-bit Linux"
                ;;
            esac
        ;;
        darwin*)
            case $(uname -m) in
                x86_64)
                    echo "darwin64"
                ;;
                *)
                    fail_unsupported "32-bit Macs"
                ;;
            esac
        ;;
        msys*)
            fail_unsupported "Windows"
        ;;
        *)
            fail_unsupported
        ;;
    esac
}

function main() {
    PLATFORM="$(detect_platform)"
    BINARY_URL="https://storage.googleapis.com/asciiroulette/ascii_roulette.$PLATFORM"
    
    echo "Downloading ASCII Roulette..."
    curl -fs "$BINARY_URL" > /tmp/ascii_roulette
    chmod +x /tmp/ascii_roulette
    clear
    
    /tmp/ascii_roulette
    
    rm /tmp/ascii_roulette
    reset
}

main
