# Watcher: Command Execution Monitor with Pause/Resume Control

## Overview

**Watcher** is a tool to execute command-line programs, monitor their output, and control their execution with pause, resume, and quit functionality. It also logs the command's output to a file for later review.

The tool runs the command in a pseudo-terminal (PTY) and logs all output while allowing the user to pause or resume the execution and terminate it when necessary. It ensures proper signal handling and provides a user-friendly interface for process control.

## Features

- **Command Execution**: Runs any shell command, capturing its output.
- **Pause/Resume**: Allows pausing the running process and resuming it when needed.
- **Quit**: Provides a way to terminate the process safely.
- **Logging**: Logs all output to a file in a structured format based on the current date.
- **Configuration**: Configurable log directory through ~/.watcher.conf.

## Installation

Clone or download the repository.

Initialize the Go module and install the required dependencies:

```
go mod init watcher
go get github.com/creack/pty
go get github.com/fatih/color
```

Build the program:

```
go build -o watcher watcher.go
```

## Usage

Run the watcher program with any command you'd like to monitor:

```
watcher <command>
```

### Examples:

- To run a curl request:

    ```
    watcher curl https://example.com
    ```

- To run gobuster:

    ```
    watcher gobuster dir -w /path/to/wordlist.txt -u http://example.com
    ```

### Command Line Options:
- `-h`, `--help`: Displays the help message and exits.

### Controls During Execution:
- **Pause**: Press `P` + Enter to pause the process.
- **Resume**: Press `R` + Enter to resume the process.
- **Quit**: Press `Q` + Enter to quit and terminate the process.

## Configuration

Watcher reads its configuration from the file `~/.watcher.conf`. If the file does not exist, it will be created with default values.

### Default Configuration:

```
log_base=/tmp/watcher_logs
```

This file allows you to specify the base directory where logs will be stored. The logs are saved in subdirectories named based on the current date.

## Logging

Logs are saved in the folder specified in the configuration file, under subfolders for each date. The log files are named after the command being executed, followed by `.log`. For example, if you run curl https://example.com on September 6, 2024, the log file will be located at:

```
/tmp/watcher_logs/6_September_2024/curl.log
```

The logs contain timestamps and records for all actions and command outputs.

