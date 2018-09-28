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
	if (len(os.Args) != 8) {
		fmt.Printf("Please issue command in the following format: command <AIP install location> <Extend Server URL> <User ID> <Password> <upgrade|install> <official|all> <stable|latest>\n") 
		fmt.Println("Example: downloadUpgradedExtensions.exe \"C:\\Program Files\\Cast\\8.3.3\" \"https://extend.castsoftware.com:443/V2/api/v2\" p.rabinovich@castsoftware.com xxxxxx upgrade all stable")
		os.Exit(1)
	}

	// Get parameterrs passed to the 
	aipDir := os.Args[1]
	serverURL := os.Args[2]
	extUsr := os.Args[3]
	extPass := os.Args[4]
	
	// upgrade|install
	upgradeOrInstall := os.Args[5]
	if !strings.EqualFold(upgradeOrInstall, "upgrade") && !strings.EqualFold(upgradeOrInstall, "install") {
		fmt.Println("Incorrect parameter specified. Please provide one of the following options: upgrade|install")
		os.Exit(1)
	}
	// official|all
	officialOrAll := os.Args[6]
	if !strings.EqualFold(officialOrAll, "official") && !strings.EqualFold(officialOrAll, "all") {
		fmt.Println("Incorrect parameter specified. Please provide one of the following options: official|all")
		os.Exit(1)
	}
	// stable|latest
	stableOrLatest := os.Args[7]
	if !strings.EqualFold(stableOrLatest, "stable") && !strings.EqualFold(stableOrLatest, "latest") {
		fmt.Println("Incorrect parameter specified. Please provide one of the following options: stable|latest")
		os.Exit(1)
	}
	
	// Check if folder location provided is valid
	if _, err := os.Stat(aipDir); os.IsNotExist(err) {
		fmt.Printf("Specified AIP directory location is invalid: %s\n", aipDir)
		fmt.Printf("Please verify and correct\n")
		os.Exit(1)
	}
	
	// Get list of upgradeable extensions or all available based on defined params
	cmd := exec.Command("foo1")
	if upgradeOrInstall == "upgrade" {
		fmt.Printf("Checking for upgradable extensions...\n")
		cmd = exec.Command(aipDir + "\\ExtensionDownloader.exe", "--server", serverURL, "--username", extUsr, "--password", extPass, "list", "upgradable")
	} else {
		fmt.Printf("Checking for available extensions...\n")
		cmd = exec.Command(aipDir + "\\ExtensionDownloader.exe", "--server", serverURL, "--username", extUsr, "--password", extPass, "list", "available")
	}
	
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
	for _, extFull := range extVerList {
		// Split extension informatino returned into extensino id and version
		extInfo := strings.Fields(extFull)
		if len(extInfo) == 2 {
			//fmt.Printf("Processing extension %s\n", extFull)
			extID := extInfo[0]
			extVer := extInfo[1]
			cmd := exec.Command("cmd", "/c")
			skipFlag := false
			// Check if installing offical or all extensions
			if officialOrAll == "official" {
				// Check if author of extenion is CASTLabs or CASTUserCommunity 
				if strings.Contains(extID, "com.castsoftware.labs.") || strings.Contains(extID, "com.castsoftware.uc.") {
					fmt.Printf("Not an offical CAST extension... Skipping: %s\n", extFull)
					skipFlag = true
				}
			}
			
			// Install latest or stable version based on passed argument
			if !skipFlag && stableOrLatest == "stable" {
				// skip if the extension is Alpha or Beta and stableOnly flag is set
				if strings.Contains(strings.ToLower(extVer), "alpha") || strings.Contains(strings.ToLower(extVer), "beta") {
					fmt.Printf("Unstable version of extension... Skipping: %s\n", extFull)
					skipFlag = true
				} else {
					fmt.Printf("Installing extension: %s\n", extFull)
					cmd = exec.Command(aipDir + "\\ExtensionDownloader.exe", "--server", serverURL, "--username", extUsr, "--password", extPass, "install", extID, "--version", extVer)
				}	
			} else if !skipFlag && stableOrLatest == "latest" {
				// If extension is not already on the list
				if !contains(extList, extID) {

					// Add to the list first
					extList = append(extList, extID)
					fmt.Printf("Upgrading extension: %s to latest available version\n", extID)
					cmd = exec.Command(aipDir + "\\ExtensionDownloader.exe", "--server", serverURL, "--username", extUsr, "--password", extPass, "install", extID)
				} else {
					fmt.Printf("Latest version of this extension is already installed: %s\n", extID)
					skipFlag = true
				}
				
			}
			
			// Only execute upgrade command if skip flag is not set
			if !skipFlag {
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
}