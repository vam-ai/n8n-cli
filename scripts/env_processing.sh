#!/bin/bash

MODE=""
INPUT_FILE=""
OUTPUT_FILE=""
ROOT_DIR="$(dirname "$(realpath "$0")")"
PASSWORD_SCRIPT="$ROOT_DIR/password_helper-aes.sh"

usage() {
    echo "Usage: $0 [-e|-d] -i input_file -o output_file"
    echo "  -e          Encrypt mode (plaintext -> ENC())"
    echo "  -d          Decrypt mode (ENC() -> plaintext)"
    echo "  -i FILE     Input file"
    echo "  -o FILE     Output file"
    echo "  -h          Show this help"
    exit 1
}

validate_dependencies() {
    if [ ! -f "$PASSWORD_SCRIPT" ]; then
        echo "Error: Password helper script not found at $PASSWORD_SCRIPT" >&2
        exit 1
    fi
    
    if ! command -v openssl &> /dev/null; then
        echo "Error: OpenSSL is required but not installed" >&2
        exit 1
    fi
}

process_line() {
    local line="$1"
    
    # Preserve comments and empty lines
    if [[ "$line" =~ ^# ]] || [[ -z "$line" ]]; then
        echo "$line" >> "$OUTPUT_FILE"
        return
    fi
    
    # Preserve empty values (key= with optional whitespace)
    if [[ "$line" =~ ^[^=]+=[[:space:]]*$ ]]; then
        echo "$line" >> "$OUTPUT_FILE"
        return
    fi
    
    case $MODE in
        encrypt)
            if [[ "$line" =~ ^([^=]+)=(.*) ]]; then
                local key="${BASH_REMATCH[1]}"
                local value="${BASH_REMATCH[2]}"
                echo "Encrypting $key..." >&2
                encrypted_value=$("$PASSWORD_SCRIPT" -e "$value")
                echo "$key=$encrypted_value" >> "$OUTPUT_FILE"
            else
                echo "$line" >> "$OUTPUT_FILE"
            fi
            ;;
            
        decrypt)
            if [[ "$line" =~ ^([^=]+)=ENC\((.*)\) ]]; then
                local key="${BASH_REMATCH[1]}"
                local encrypted_value="ENC(${BASH_REMATCH[2]})"
                echo "Decrypting $key..." >&2
                decrypted_value=$("$PASSWORD_SCRIPT" -d "$encrypted_value")
                echo "$key=$decrypted_value" >> "$OUTPUT_FILE"
            else
                echo "$line" >> "$OUTPUT_FILE"
            fi
            ;;
    esac
}

main() {
    validate_dependencies
    echo "" > "$OUTPUT_FILE"

    while IFS= read -r line; do
        process_line "$line"
    done < "$INPUT_FILE"

    echo "Operation completed successfully"
    echo "Input:  $(realpath "$INPUT_FILE")"
    echo "Output: $(realpath "$OUTPUT_FILE")"
}

# Argument parsing and validation remains the same as previous version
# [Include the corrected argument parsing from earlier solution here]

while getopts "edi:o:h" opt; do
    case $opt in
        e) MODE="encrypt" ;;
        d) MODE="decrypt" ;;
        i) INPUT_FILE="$OPTARG" ;;
        o) OUTPUT_FILE="$OPTARG" ;;
        h) usage ;;
        *) usage ;;
    esac
done

if [ -z "$MODE" ] || [ -z "$INPUT_FILE" ] || [ -z "$OUTPUT_FILE" ]; then
    echo "Error: Missing required arguments" >&2
    usage
fi

if [ ! -f "$INPUT_FILE" ]; then
    echo "Error: Input file $INPUT_FILE not found" >&2
    exit 1
fi

if [ "$INPUT_FILE" == "$OUTPUT_FILE" ]; then
    echo "Error: Input and output files must be different" >&2
    exit 1
fi

main