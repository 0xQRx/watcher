package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/fatih/color"
)

var logFile *os.File

// Get current date as formatted string for folder names
func getCurrentDateFolder() string {
	now := time.Now()
	return fmt.Sprintf("%d_%s_%d", now.Day(), now.Month().String(), now.Year())
}

// Create date folder and return log file path for the program
func createLogFile(baseFolder, programName string) (*os.File, error) {
	dateFolder := filepath.Join(baseFolder, getCurrentDateFolder())
	err := os.MkdirAll(dateFolder, 0755) // Ensure the folder exists
	if err != nil {
		return nil, err
	}

	logFilePath := filepath.Join(dateFolder, programName+".log")
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return logFile, nil
}

// Read configuration file ~/.watcher.conf
func readConfig() (map[string]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	configPath := filepath.Join(homeDir, ".watcher.conf")
	config := make(map[string]string)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("Config file not found. Creating default.")
		err := os.WriteFile(configPath, []byte("log_base=/tmp/watcher_logs\n"), 0644)
		if err != nil {
			return nil, err
		}
		config["log_base"] = "/tmp/watcher_logs"
		return config, nil
	}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "=") {
			parts := strings.Split(line, "=")
			config[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return config, nil
}

// Start a command in a PTY and return its output reader
func startCommandInPTY(command string) (*exec.Cmd, *os.File, error) {
	cmd := exec.Command("bash", "-c", command)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, nil, err
	}
	return cmd, ptmx, nil
}

// Log actions with timestamp
func logAction(action string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	log.SetOutput(logFile)
	log.Printf("[%s] %s\n", timestamp, action)
}

// Print the ASCII banner and controls with colors
func printBanner() {
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()

	fmt.Println(red(`
 __      __  _________________________   ___ ________________________ 
/  \    /  \/  _  \__    ___/\_   ___ \ /   |   \_   _____/\______   \
\   \/\/   /  /_\  \|    |   /    \  \//    ~    \    __)_  |       _/
 \        /    |    \    |   \     \___\    Y    /        \ |    |   \
  \__/\  /\____|__  /____|    \______  /\___|_  /_______  / |____|_  /
       \/         \/                 \/       \/        \/         \/  
`))

	fmt.Println(blue("Controls during execution:"))
	fmt.Println(green("  Press 'P' + Enter to pause the execution."))
	fmt.Println(green("  Press 'R' + Enter to resume the execution."))
	fmt.Println(green("  Press 'Q' + Enter to quit and terminate the process."))
}

// Stream process output from the PTY to both console and log file
func streamPTYOutput(ptmx *os.File, logFile *os.File, done chan bool) {
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	scanner := bufio.NewScanner(ptmx)

	// Print the ASCII banner and controls before command output
	printBanner()

	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintln(multiWriter, line) // Write command output to console and log
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		if strings.Contains(err.Error(), "input/output error") {
			return
		}
		fmt.Printf("Error reading PTY output: %v\n", err)
	}

	done <- true
}

// Wait for user input and control the process
func controlProcess(cmd *exec.Cmd, done chan bool) {
	reader := bufio.NewReader(os.Stdin)

	for {
		char, _ := reader.ReadString('\n')
		switch strings.TrimSpace(strings.ToLower(char)) {
		case "p":
			fmt.Println("Pausing the process... (Press 'R' + Enter to resume or 'Q' + Enter to quit)")
			syscall.Kill(-cmd.Process.Pid, syscall.SIGSTOP)
			logAction("Process paused")
		case "r":
			fmt.Println("Resuming the process... (Press 'P' + Enter to pause or 'Q' + Enter to quit)")
			syscall.Kill(-cmd.Process.Pid, syscall.SIGCONT)
			logAction("Process resumed")
		case "q":
			fmt.Println("Quitting...")
			cmd.Process.Kill()
			logAction("Process terminated")
			done <- true // Signal the watcher to exit
			return
		}
	}
}

// Display help menu
func displayHelp() {
	helpText := `
Usage: watcher <command> [options]

Watcher is a program to monitor and control the execution of a command-line tool.

Options:
  -h, --help    Show this help message and exit.

Examples:
  watcher curl https://example.com
  watcher gobuster dir -w /path/to/wordlist.txt -u http://example.com

Controls during execution:
  Press 'P' + Enter to pause the execution.
  Press 'R' + Enter to resume the execution.
  Press 'Q' + Enter to quit and terminate the process.


Logs are stored in a configurable location: ~/.watcher.conf
`
	fmt.Println(helpText)
}

func main() {
	// Check for -h or --help flag and display help if found
	helpFlag := flag.Bool("h", false, "Show help")
	flag.Parse()

	if *helpFlag || len(os.Args) < 2 {
		displayHelp()
		return
	}

	// Read the config
	config, err := readConfig()
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		return
	}

	// Open log file based on date and program name
	command := strings.Join(flag.Args(), " ")
	baseFolder := config["log_base"]
	programName := strings.Split(filepath.Base(flag.Args()[0]), " ")[0] // Extract program name
	logFile, err = createLogFile(baseFolder, programName)
	if err != nil {
		fmt.Printf("Error creating log file: %v\n", err)
		return
	}
	defer logFile.Close()

	// Start the command in a PTY
	cmd, ptmx, err := startCommandInPTY(command)
	if err != nil {
		fmt.Printf("Error starting command: %v\n", err)
		return
	}
	defer ptmx.Close()

	logAction(fmt.Sprintf("Started command: %s", command))

	// Catch interrupt signals to handle process cleanly
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logAction("Received interrupt signal, terminating process")
		cmd.Process.Kill()
		os.Exit(0)
	}()

	// Stream PTY output and control process
	done := make(chan bool)
	go streamPTYOutput(ptmx, logFile, done)
	go controlProcess(cmd, done)

	// Wait for the command to finish or user input to quit
	go func() {
		err = cmd.Wait()
		done <- true
	}()

	// Wait for either the user or the process to finish
	<-done

	if err != nil {
		logAction(fmt.Sprintf("Command finished with error: %v", err))
	} else {
		logAction("Command finished successfully")
	}
}
