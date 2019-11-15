package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/malashin/ffinfo"
	"github.com/wlredeye/jsonlines"
)

// Input file must have "UUID\tURL" structure.

var inputPath string = "input.txt"
var databasePath string = "database.json"
var errorLogPath string = "errorlog.log"

var dbSlice []File
var dbMap = make(map[string]File)

var re *regexp.Regexp = regexp.MustCompile(`([a-z0-9]{32})\t(.+)`)

type UuidURL struct {
	UUID string
	URL  string
}

type File struct {
	UuidURL
	Probe ffinfo.File
}

func main() {
	// Read database file and fill up the database map.
	if _, err := os.Stat(databasePath); err == nil {
		file, err := os.Open(databasePath)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		err = jsonlines.Decode(file, &dbSlice)
		if err != nil {
			panic(err)
		}

		for _, entry := range dbSlice {
			dbMap[entry.UUID] = entry
		}
	}

	// Read input file.
	lines, err := readLines(inputPath)
	if err != nil {
		panic(err)
	}

	// Create new error logger.
	errorLog, err := os.OpenFile(errorLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0775)
	if err != nil {
		panic(err)
	}
	defer errorLog.Close()

	Error := log.New(errorLog, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	total := len(lines)

	for i, line := range lines {
		fmt.Printf("%v/%v: ", i+1, total)

		id, err := parseLine(line)
		if err != nil {
			fmt.Println("ERROR:", err)
			Error.Println(fmt.Sprintf("%v: %v\n", line, err))
			continue
		}

		if _, ok := dbMap[id.UUID]; ok {
			fmt.Println("skiping", id.URL)
			continue
		}

		f := File{
			UuidURL: id,
		}

		p, err := ffinfo.Probe(f.URL)
		if err != nil {
			fmt.Println("ERROR:", err)
			Error.Println(fmt.Sprintf("%v %v: %v", f.UUID, f.URL, err))
			continue
		}

		f.Probe = *p
		dbMap[f.UUID] = f

		fmt.Printf("%v\n", f.URL)

		file, err := os.OpenFile(databasePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		err = jsonlines.Encode(file, &[]File{f})
		if err != nil {
			panic(err)
		}
	}
}

func parseLine(line string) (id UuidURL, err error) {
	if !re.MatchString(line) {
		return id, fmt.Errorf("input line does not match \"UUDI\tURL\" pattern: %v", line)
	}

	match := re.FindStringSubmatch(line)

	id = UuidURL{
		UUID: match[1],
		URL:  match[2],
	}

	return id, nil
}

func writeStringToFile(filename string, str string) error {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0775)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.WriteString(str); err != nil {
		return err
	}

	return nil
}

// readLines reads a whole file into memory
// and returns a slice of its lines.
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}
