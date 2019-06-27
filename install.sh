#!/bin/sh

echo "Downloading ASCII Roulette..."
curl -fs https://storage.googleapis.com/asciiroulette/asciirtc > /tmp/asciirtc
chmod +x /tmp/asciirtc
clear

/tmp/asciirtc

rm /tmp/asciirtc
