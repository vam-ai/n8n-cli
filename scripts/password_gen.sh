#!/bin/bash

# Random Password Generator

DEFAULT_LENGTH=16
CHAR_SET="A-Za-z0-9_-"

usage() {
    echo "Usage: $0 [-l length]"
    echo "Generate random password with specified length (default: $DEFAULT_LENGTH)"
    exit 0
}

# Parse arguments
while getopts ":l:h" opt; do
    case $opt in
        l)
            length="$OPTARG"
            ;;
        h)
            usage
            ;;
        *)
            echo "Invalid option: -$OPTARG" >&2
            exit 1
            ;;
    esac
done

# Validate length
length=${length:-$DEFAULT_LENGTH}
if ! [[ "$length" =~ ^[0-9]+$ ]] || [ "$length" -lt 1 ]; then
    echo "Error: Password length must be a positive integer" >&2
    exit 1
fi

# Generate password
password=$(LC_ALL=C tr -dc "$CHAR_SET" < /dev/urandom | head -c "$length")

echo "Generated password ($length characters):"
echo "$password"