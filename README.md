# Patch-SEB

Patch-SEB is a tool designed to patch Safe Exam Browser (SEB) on Windows. This tool downloads the latest release of SEB patches and applies them to the SEB installation directory.

## Usage

1. Download the latest release.
2. Run as administrator
3. Profit

## Building from Source

To build the project yourself, follow these steps:

1. **Clone the Repository**:
    ```sh
    git clone https://github.com/wxnnvs/patch-seb.git
    cd patch-seb
    ```

2. **Install Dependencies**:
    Ensure you have Go installed. Then, run:
    ```sh
    go mod tidy
    ```

3. **Build the Executable**:
    ```sh
    go build -o patch-seb.exe main.go
    ```

## Important Note

This tool might get detected as a trojan by some antivirus software due to its nature of modifying files in the SEB installation directory. Ensure you understand the risks and have appropriate permissions to use this tool.

# License
Do Whatever The Fuck You Want License