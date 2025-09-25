package models

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/uuid"
)

type SemgrepItem struct {
	MD5      string
	RuleID   string
	From     string
	To       string
	Severity string
}
type SemgrepResult struct {
	Issues  []SemgrepItem
	Purl    string
	Version string
}

type URLItem struct {
	URLHash  string
	PurlName string
	Version  string
}
type PivotItem struct {
	URLHash  string
	FileHash string
}

var LDBFileTableName string
var LDBSemgreptTableName string
var LDBPivotTableName string
var LDBBinPath string
var LDBEncBinPath string

// Checks if the LBD exists and returns the list of available tables.
func PingLDB(ldbname string) ([]string, error) {
	var ret []string
	entry, err := os.ReadDir("/var/lib/ldb/" + ldbname)
	if err != nil {
		return []string{}, errors.New("Problems opening LDB " + ldbname)
	}
	for e := range entry {
		if entry[e].IsDir() {
			ret = append(ret, ldbname+"/"+entry[e].Name())
		}
	}

	return ret, nil
}

// Single item worker for Semgrep. From a MD5 of a file enqueues a list of SemgrepItem

func QueryBulkPivotLDB(keys []string) map[string][]string {
	ret := make(map[string][]string)

	name := fmt.Sprintf("/tmp/%s-pivot.txt", uuid.New().String())
	f, err := os.Create(name)
	if err != nil {
		return map[string][]string{}
	}
	var written = 0
	for job := range keys {
		if keys[job] != "" {
			line := fmt.Sprintf("select from %s key %s csv hex 32\n", LDBPivotTableName, keys[job])
			n, err := f.WriteString(line)
			if err == nil {
				written += n
			}
		}
	}
	f.Close()
	if written > 0 {
		ldbCmd := exec.Command(LDBBinPath, "-f", name)

		buffer, errLDB := ldbCmd.Output()
		if errLDB != nil {
			fmt.Println(errLDB)
		}
		// split results line by line
		// each row contains 3 values: <UrlMD5>,<FileMD5>,unknown
		lines := strings.Split(string(buffer), "\n")

		for i := range lines {
			fields := strings.Split(lines[i], ",")
			if len(fields) == 3 {
				ret[fields[0]] = append(ret[fields[0]], fields[1])
			}
		}
	}
	os.Remove(name)
	return ret
}

func QueryBulkSemgrepLDB(items map[string][]string) map[string][]SemgrepItem {
	issues := make(map[string][]SemgrepItem)

	name := fmt.Sprintf("/tmp/%s-semgrep.txt", uuid.New().String())
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE, 0600)

	if err != nil {
		return map[string][]SemgrepItem{}
	}
	added := make(map[string]bool)
	var written = 0
	for job := range items {
		fileHashes := items[job]
		for r := range fileHashes {
			if _, exist := added[fileHashes[r]]; !exist {
				line := fmt.Sprintf("select from %s key %s csv hex 16\n", LDBSemgreptTableName, fileHashes[r])
				n, err := f.WriteString(line)
				added[fileHashes[r]] = true
				if err == nil {
					written += n
				}
			}
		}
	}
	f.Close()
	if written > 0 {
		ldbCmd := exec.Command(LDBBinPath, "-f", name)
		buffer, _ := ldbCmd.Output()
		lines := strings.Split(string(buffer), "\n")

		for i := range lines {
			fields := strings.Split(lines[i], ",")
			if len(fields) == 5 {
				issue := SemgrepItem{MD5: fields[0], RuleID: fields[1], From: fields[2], To: fields[3], Severity: fields[4]}
				issues[fields[0]] = append(issues[fields[0]], issue)
			}
		}
	}
	os.Remove(name)
	return issues
}

func ContainsTable(arr []string, value string) bool {
	for r := range arr {
		if arr[r] == value {
			return true
		}
	}
	return false
}

func QueryBulkFileLDB(fileURL []string) map[string]string {
	name := fmt.Sprintf("/tmp/%s-file.txt", uuid.New().String())
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return map[string]string{}
	}
	var written = 0
	for job := range fileURL {
		reqFields := strings.Split(fileURL[job], "-")
		if len(reqFields) == 2 {
			line := fmt.Sprintf("select from %s key %s csv hex 32\n", LDBFileTableName, reqFields[0])
			n, err := f.WriteString(line)
			if err == nil {
				written += n
			}
		}
	}

	f.Close()
	if written > 0 {
		ldbCmd := exec.Command(LDBEncBinPath, "-f", name)
		buffer, _ := ldbCmd.Output()
		res := make(map[string]string)
		// split results line by line
		// each row contains 3 values: <UrlMD5>,<FileMD5>,unknown
		lines := strings.Split(string(buffer), "\n")

		for i := range lines {
			fields := strings.Split(lines[i], ",")
			if len(fields) == 3 {
				// fmt.Println(lines[i])
				k := fields[0] + "-" + fields[1]
				v := fields[2]
				res[k] = v
			}
		}

		ret := make(map[string]string)
		for r := range fileURL {
			if v, exists := res[fileURL[r]]; exists {
				pair := strings.Split(fileURL[r], "-")
				ret[pair[0]] = v
			}
		}

		os.Remove(name)
		return ret
	}
	return map[string]string{}
}
