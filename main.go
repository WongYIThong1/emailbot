package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

// ANSI Color codes
const (
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	Bold    = "\033[1m"
)

const (
	usageFilePath = "data/usage.enc"
)

var encryptionKey = []byte("0123456789ABCDEF0123456789ABCDEF")

// ASCII Art for "EDITOR BOT"
func printBanner() {
	banner := `
    ███████╗██████╗ ██╗████████╗ ██████╗ ██████╗ 
    ██╔════╝██╔══██╗██║╚══██╔══╝██╔═══██╗██╔══██╗
    ██║     ██║  ██║██║   ██║   ██║   ██║██████╔╝
    ██║     ██║  ██║██║   ██║   ██║   ██║██╔══██╗
    ╚██████╗██████╔╝██║   ██║   ╚██████╔╝██║  ██║
     ╚═════╝╚═════╝ ╚═╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝
                                                
              ██████╗  ██████╗ ████████╗
              ██╔══██╗██╔═══██╗╚══██╔══╝
              ██████╔╝██║   ██║   ██║   
              ██╔══██╗██║   ██║   ██║   
              ██████╔╝╚██████╔╝   ██║   
              ╚═════╝  ╚═════╝    ╚═╝   
                                        
         Combo Editing Bot - Email Data Generator
`
	fmt.Print(Cyan + Bold + banner + Reset)
}

func ensureUsageStorage() error {
	return os.MkdirAll(filepath.Dir(usageFilePath), os.ModePerm)
}

func formatK(value int64) string {
	k := int64(math.Round(float64(value) / 1000.0))
	if k < 0 {
		k = 0
	}
	return fmt.Sprintf("%dk", k)
}

func readUsageCount() int64 {
	if err := ensureUsageStorage(); err != nil {
		fmt.Printf(Yellow+"[WARN] "+Reset+"Unable to prepare usage storage: %v\n", err)
		return 0
	}

	data, err := os.ReadFile(usageFilePath)
	if err != nil {
		return 0
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		fmt.Printf(Yellow+"[WARN] "+Reset+"Unable to init cipher: %v\n", err)
		return 0
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		fmt.Printf(Yellow+"[WARN] "+Reset+"Unable to init GCM: %v\n", err)
		return 0
	}

	nonceSize := gcm.NonceSize()
	if len(data) <= nonceSize {
		return 0
	}

	nonce := data[:nonceSize]
	ciphertext := data[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		fmt.Printf(Yellow+"[WARN] "+Reset+"Failed to decrypt usage counter: %v\n", err)
		return 0
	}

	value, err := strconv.ParseInt(string(plaintext), 10, 64)
	if err != nil {
		fmt.Printf(Yellow+"[WARN] "+Reset+"Failed to parse usage counter: %v\n", err)
		return 0
	}

	return value
}

func writeUsageCount(total int64) error {
	if err := ensureUsageStorage(); err != nil {
		return err
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(crand.Reader, nonce); err != nil {
		return err
	}

	plaintext := []byte(strconv.FormatInt(total, 10))
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	payload := append(nonce, ciphertext...)
	return os.WriteFile(usageFilePath, payload, 0600)
}

func updateUsageCount(previous, delta int64) int64 {
	newTotal := previous + delta
	if err := writeUsageCount(newTotal); err != nil {
		fmt.Printf(Yellow+"[WARN] "+Reset+"Failed to persist AI training counter: %v\n", err)
	}
	return newTotal
}

// Enable ANSI color support on Windows
func enableColors() {
	handle := windows.Handle(os.Stdout.Fd())
	var mode uint32
	windows.GetConsoleMode(handle, &mode)
	mode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	windows.SetConsoleMode(handle, mode)
}

// Define modification patterns
func getModificationPatterns() (prefixes, suffixes, specialChars []string) {
	// Numeric prefixes/suffixes
	prefixes = []string{
		"123", "456", "789", "001", "007", "999",
		"2024", "2025", "2023", "2022",
		"01", "12", "24", "99",
		"abc", "xyz", "new", "old",
		"mr", "ms", "my", "the",
		"123abc", "abc123", "2024new", "new2024",
	}

	suffixes = []string{
		"123", "456", "789", "001", "007", "999",
		"2024", "2025", "2023", "2022",
		"01", "12", "24", "99",
		"abc", "xyz", "new", "old",
		"mail", "email", "user", "test",
		"123abc", "abc123", "2024new", "new2024",
	}

	// Special character patterns
	specialChars = []string{
		".", "_", "-",
		"+work", "+news", "+shop", "+mail", "+test",
	}

	return prefixes, suffixes, specialChars
}

// Randomly select an element from a list
func randomSelect(list []string) string {
	if len(list) == 0 {
		return ""
	}
	return list[rand.Intn(len(list))]
}

// Open file selection dialog using PowerShell
func selectFile() (string, error) {
	cmd := exec.Command("powershell", "-Command", `Add-Type -AssemblyName System.Windows.Forms

$f = New-Object System.Windows.Forms.OpenFileDialog
$f.Filter = 'Text files (*.txt)|*.txt'
if ($f.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
    $f.FileName
}`)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error selecting file: %v", err)
	}

	filename := strings.TrimSpace(string(output))
	if filename == "" {
		return "", fmt.Errorf("no file selected")
	}

	return filename, nil
}

// Validate email:password format
func validateFormat(line string) bool {
	// Remove leading/trailing whitespace
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}

	// Regex pattern: email:password format
	// email format: allows letters, numbers, dots, underscores, hyphens, plus signs
	// password format: allows any non-whitespace characters
	pattern := `^[a-zA-Z0-9._+\-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}:[^\s]+$`
	matched, err := regexp.MatchString(pattern, line)
	if err != nil {
		return false
	}

	return matched
}

// Parse email address
func parseEmail(email string) (username, domain string, err error) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid email format: %s", email)
	}
	return parts[0], parts[1], nil
}

// Generate special character variant
func generateSpecialCharVariant(username, domain, password, specialChar string) string {
	if strings.HasPrefix(specialChar, "+") {
		// Plus alias pattern: username+alias@domain
		return fmt.Sprintf("%s%s@%s:%s", username, specialChar, domain, password)
	} else {
		// Other special characters: insert in the middle of username
		// If username length > 1, insert at middle; otherwise append
		if len(username) > 1 {
			mid := len(username) / 2
			newUsername := username[:mid] + specialChar + username[mid:]
			return fmt.Sprintf("%s@%s:%s", newUsername, domain, password)
		} else {
			return fmt.Sprintf("%s%s@%s:%s", username, specialChar, domain, password)
		}
	}
}

// Generate 3 random data entries for each email
func generateData(email, password string) []string {
	username, domain, err := parseEmail(email)
	if err != nil {
		return []string{}
	}

	prefixes, suffixes, specialChars := getModificationPatterns()
	data := []string{}

	// 1. Prefix variant
	prefix := randomSelect(prefixes)
	if prefix != "" {
		data = append(data, fmt.Sprintf("%s%s@%s:%s", prefix, username, domain, password))
	}

	// 2. Suffix variant
	suffix := randomSelect(suffixes)
	if suffix != "" {
		data = append(data, fmt.Sprintf("%s%s@%s:%s", username, suffix, domain, password))
	}

	// 3. Special character variant
	specialChar := randomSelect(specialChars)
	if specialChar != "" {
		data = append(data, generateSpecialCharVariant(username, domain, password, specialChar))
	}

	return data
}

// Create output directory with timestamp
func createOutputDir() (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	outputDir := filepath.Join("output", timestamp)

	err := os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}

	return outputDir, nil
}

// Display error message
func showError(message string) {
	fmt.Printf(Red+Bold+"[ERROR] "+Reset+Red+"%s\n"+Reset, message)
}

// Process file and generate output
func processFile(filename string) (int64, error) {
	// Open input file
	file, err := os.Open(filename)
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Create output directory
	fmt.Print(Yellow + "[INFO] " + Reset + "Creating output directory...\n")
	outputDir, err := createOutputDir()
	if err != nil {
		return 0, err
	}
	fmt.Print(Green + "[SUCCESS] " + Reset + "Output directory created: " + Cyan + outputDir + Reset + "\n\n")

	// Create output file
	outputPath := filepath.Join(outputDir, "output.txt")
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	writer := bufio.NewWriter(outputFile)

	// Read and process file content
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 1024)
	scanner.Buffer(buf, 10*1024*1024)
	lineNum := 0
	processedCount := int64(0)
	errorCount := 0
	resultsSet := make(map[string]struct{})
	results := make([]string, 0, 1024)

	addEntry := func(entry string) {
		if _, exists := resultsSet[entry]; exists {
			return
		}
		resultsSet[entry] = struct{}{}
		results = append(results, entry)
		processedCount = int64(len(results))
	}

	fmt.Print(Cyan + Bold + "[AI EDIT METHOD] " + Reset + "Processing data stream... the model is learning in real time.\n\n")

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// Validate format
		if !validateFormat(line) {
			fmt.Printf(Yellow+"[WARN] "+Reset+"Line %d: Invalid format, skipped: %s\n", lineNum, line)
			errorCount++
			continue
		}

		// Parse email:password
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			fmt.Printf(Yellow+"[WARN] "+Reset+"Line %d: Format error, skipped: %s\n", lineNum, line)
			errorCount++
			continue
		}

		email := parts[0]
		password := parts[1]

		// Generate data
		data := generateData(email, password)
		for _, entry := range data {
			addEntry(entry)
		}

		// Progress indicator
		if lineNum%10 == 0 {
			fmt.Printf(Green+"[PROGRESS] "+Reset+"Processed %d lines, curated %d AI data entries...\r", lineNum, processedCount)
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading file: %v", err)
	}

	rand.Shuffle(len(results), func(i, j int) {
		results[i], results[j] = results[j], results[i]
	})

	for _, entry := range results {
		if _, err := writer.WriteString(entry + "\n"); err != nil {
			return 0, fmt.Errorf("error writing to file: %v", err)
		}
	}

	if err := writer.Flush(); err != nil {
		return 0, fmt.Errorf("error flushing output: %v", err)
	}

	fmt.Print("\n\n")
	fmt.Print(Green + Bold + "Processing Complete!\n\n" + Reset)

	fmt.Printf(Green+"[SUCCESS] "+Reset+"Output file: "+Cyan+Bold+"%s\n"+Reset, outputPath)
	fmt.Printf(Green+"[SUCCESS] "+Reset+"Unique AI data entries generated: "+Yellow+Bold+"%d\n"+Reset, processedCount)
	if errorCount > 0 {
		fmt.Printf(Yellow+"[WARN] "+Reset+"Skipped lines: "+Red+Bold+"%d\n"+Reset, errorCount)
	}

	return processedCount, nil
}

func main() {
	// Enable ANSI color support on Windows
	enableColors()

	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	// Clear screen and print banner
	cmd := exec.Command("cmd", "/c", "cls")
	cmd.Stdout = os.Stdout
	cmd.Run()

	printBanner()
	fmt.Print("\n" + Magenta + Bold + "[AI EDIT METHOD] " + Reset + "This editor feeds an in-house AI that adapts to every combo you import.\n")
	fmt.Print(Magenta + Bold + "[AI ADVICE] " + Reset + "Upload massive datasets—more data means a sharper edit pattern engine.\n\n")

	totalImported := readUsageCount()
	fmt.Printf(Cyan+Bold+"[AI POWER] "+Reset+"Total data imported so far: "+Yellow+Bold+"%s\n\n"+Reset, formatK(totalImported))
	fmt.Print(Cyan + Bold + "Please select a file to process..." + Reset + "\n\n")

	// Select file
	filename, err := selectFile()
	if err != nil {
		showError(err.Error())
		fmt.Print("\n" + Yellow + "Press Enter to exit..." + Reset)
		fmt.Scanln()
		return
	}

	fmt.Printf(Green+"[SUCCESS] "+Reset+"File selected: "+Cyan+Bold+"%s\n\n"+Reset, filename)

	// Process file
	batchCount, err := processFile(filename)
	if err != nil {
		showError(err.Error())
		fmt.Print("\n" + Yellow + "Press Enter to exit..." + Reset)
		fmt.Scanln()
		return
	}

	totalImported = updateUsageCount(totalImported, batchCount)
	fmt.Printf("\n"+Magenta+"[AI TRAINING] "+Reset+"AI Editor trained with +%s data (Total: %s)\n\n", formatK(batchCount), formatK(totalImported))
	fmt.Print("\n" + Green + "All done! Thank you for using Editor Bot!\n\n" + Reset)
	fmt.Print(Yellow + "Press Enter to exit..." + Reset)
	fmt.Scanln()
}
