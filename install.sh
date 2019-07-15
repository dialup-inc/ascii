#!/bin/bash

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
# Github: https://github.com/dialup-inc/ascii
# Website: https://dialup.com/ascii
# Email: webmaster@dialup.com

function fail_unsupported() {
    platform="${1:-"your platform"}"
    echo "ASCII Roulette isn't supported on $platform yet :-( Try Mac or Linux instead."
    echo ""
    echo "Contributions are welcome! https://github.com/dialup-inc/ascii"
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

function getMD5() {
    file="$1"
    if ! [ -f "$file" ]; then
        return
    fi
    
    if [ -x "$(command -v md5sum)" ]; then
        md5sum | cut -d ' ' -f 1
        elif [ -x "$(command -v md5)" ]; then
        md5 -r $file | cut -d ' ' -f 1
    fi
}

function checkEtag() {
    url="$1"
    etag="$2"
    
    code="$(curl -s --head -o /dev/null -w "%{http_code}" -H "If-None-Match: $etag" "$url")"
    
    if [ "$code" != "304" ]; then
        echo "changed"
    fi
}

function main() {
    platform="$(detect_platform)"
    binaryURL="https://storage.googleapis.com/asciiroulette/ascii_roulette.$platform"
    md5="$(getMD5 /tmp/ascii_roulette)"
    
    if [ -n "$(checkEtag $binaryURL $md5)" ]; then
        echo -e "Downloading ASCII Roulette\e[5m...\e[0m"
        curl -fs "$binaryURL" > /tmp/ascii_roulette
    fi
    
    chmod +x /tmp/ascii_roulette
    
    clear
    /tmp/ascii_roulette
    
    if [ $? -eq 0 ]; then
        reset
    fi
}

main
