package main

import(
    "fmt"
    "os"
	"os/exec"
	"io"
	"log"
	"strings"
)

func copyAndCapture(w io.Writer, r io.Reader) ([]byte, error) {
	var out []byte
	buf := make([]byte, 1024, 1024)
	for {
		n, err := r.Read(buf[:])
		if n > 0 {
			d := buf[:n]
			out = append(out, d...)
			_, err := w.Write(d)
			if err != nil {
				return out, err
			}
		}
		if err != nil {
			// Read returns io.EOF at the end of file, which is not an error for us
			if err == io.EOF {
				err = nil
			}
			return out, err
		}
	}
	// never reached
	panic(true)
	return nil, nil
}

func contains(s []string, e string) bool {
    for _, a := range s {
        if a == e {
            return true
        }
    }
    return false
}

func main(){
	// Make sure required parameters are passed
	if (len(os.Args) != 4) {
		fmt.Printf("Please issue command in the following format: command <AIP install location> <user> <password>\n") 
		fmt.Println("Example: downloadUpgradedExtensions.exe \"C:\\Program Files\\Cast\\8.3.3\" p.rabinovich@castsoftware.com xxxxxx")
		os.Exit(1)
	}

	aipDir := os.Args[1]
	extUsr := os.Args[2]
	extPass := os.Args[3]
	
	// Check if folder location provided is valid
	if _, err := os.Stat(aipDir); os.IsNotExist(err) {
		fmt.Printf("Specified AIP directory location is invalid: %s\n", aipDir)
		fmt.Printf("Please verify and correct\n")
		os.Exit(1)
	} 
	
	fmt.Printf("Checking for upgradable extensions...\n")
	cmd := exec.Command(aipDir + "\\ExtensionDownloader.exe", "--server", "https://extend.castsoftware.com:443/V2/api/v2", "--username", extUsr, "--password", extPass, "list", "upgradable")
	var stdout, stderr []byte
	var errStdout, errStderr error
	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()
	cmd.Start()

	go func() {
		stdout, errStdout = copyAndCapture(os.Stdout, stdoutIn)
	}()

	go func() {
		stderr, errStderr = copyAndCapture(os.Stderr, stderrIn)
	}()

	err := cmd.Wait()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
	if errStdout != nil || errStderr != nil {
		log.Fatalf("failed to capture stdout or stderr\n")
	}
	
	outStr := string(stdout)
	
	extVerList := strings.Split(outStr,"\n")
	extList := []string{}
	if len(extVerList) > 1 {
		fmt.Printf("\nStarting extension upgrade process...\n")
	} else {
		fmt.Printf("\nNo new versions of installed extensions found. All is up to date.\n")
	}
	for _, element := range extVerList {
		extInfo := strings.Fields(element)
		if len(extInfo) == 2 {
			extID := extInfo[0]
			if !contains(extList, extID) {
				extList = append(extList, extID)
				fmt.Printf("Upgrading extension: %s\n", extID)
				cmd := exec.Command(aipDir + "\\ExtensionDownloader.exe", "--server", "https://extend.castsoftware.com:443/V2/api/v2", "--username", extUsr, "--password", extPass, "install", extID)
				fmt.Printf("Executing command: %s\n", cmd.Args)
				stdoutIn, _ := cmd.StdoutPipe()
				stderrIn, _ := cmd.StderrPipe()
				cmd.Start()

				go func() {
					stdout, errStdout = copyAndCapture(os.Stdout, stdoutIn)
				}()

				go func() {
					stderr, errStderr = copyAndCapture(os.Stderr, stderrIn)
				}()
				err := cmd.Wait()
				if err != nil {log.Fatalf("cmd.Run() failed with %s\n", err)}
			}
		}
	}
	
	//fmt.Printf("ext:\n%s\n", outStr)
}