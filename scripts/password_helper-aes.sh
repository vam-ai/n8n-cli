#!/bin/bash

# AES-256 Encryption/Decryption Script with ENC() Wrapper

set -eo pipefail

usage() {
    echo "Usage: $0 [-e \"plaintext\"] [-d \"ENC(base64_cipher)\"]"
    echo "  -e \"plaintext\"       Encrypt string with AES-256-CBC"
    echo "  -d \"ENC(base64_cipher)\"  Decrypt ENC() wrapped cipher"
    exit 1
}

validate_enc_format() {
    local input="$1"
    if [[ ! "$input" =~ ^ENC\(.*\)$ ]]; then
        echo "Error: Encrypted message must be in ENC() format" >&2
        exit 1
    fi
}

encrypt() {
    local plaintext="$1"
    local password
    local encrypted_output

    password=${ENC_KEY}

    encrypted_output=$(echo -n "$plaintext" | \
    openssl enc -aes-256-cbc -pbkdf2 -iter 100000 -salt -base64 -A -pass pass:"$password")

    echo "ENC($encrypted_output)"
}


decrypt() {
    local base64_cipher="$1"
    local password
    local encrypted_part

    # Validate input format
    validate_enc_format "$base64_cipher"

    # Remove ENC() wrapper
    encrypted_part="${base64_cipher#ENC(}"
    encrypted_part="${encrypted_part%)}"
    password=${ENC_KEY}

    # Perform decryption with error output
    echo -n "$encrypted_part" | \
    openssl enc -d -aes-256-cbc -pbkdf2 -iter 100000 -base64 -A -pass pass:"$password" 2>&1 || {
        echo -e "\nDecryption failed." >&2
        exit 1
    }
}

# Check for required environment variable
if [ -z "$ENC_KEY" ]; then
    echo "Error: ENC_KEY environment variable is not set" >&2
    exit 1
fi

# Parse arguments
while getopts "e:d:" opt; do
    case $opt in
        e) encrypt_text="$OPTARG";;
        d) decrypt_text="$OPTARG";;
        *) usage;;
    esac
done

# Handle encryption
if [ -n "$encrypt_text" ]; then
    encrypt "$encrypt_text"
    exit 0
fi

# Handle decryption
if [ -n "$decrypt_text" ]; then
    decrypt "$decrypt_text"
    exit 0
fi

# If no valid options were passed
usage