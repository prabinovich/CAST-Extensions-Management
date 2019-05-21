 package main

import (
	"database/sql"
	_ "github.com/lib/pq"
	"fmt"
	"strings"
	"io"
    "os"
	"os/exec"
	"log"
	"path/filepath"
	"io/ioutil"
)

const (
	//DB_USER     = "operator"
	//DB_PASSWORD = "CastAIP"
)

type extVerStruct struct {
    verFull		string
	verNum		string
    verType		string
}

// Get list of downloaded extensions and create a map of latest available versions
func enumInstalledExt (aipDir string, extMap map[string]*extVerStruct) () {
		
	fmt.Printf("\nEnumerating downloaded extensions...\n\n")
	cmd := exec.Command(aipDir + "\\ExtensionDownloader.exe", "list", "installed")
	fmt.Printf("Executing command: %s\n", cmd.Args)
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
	if err != nil {log.Fatalf("cmd.Run() failed with %s\n", err)}
	if errStdout != nil || errStderr != nil {log.Fatalf("failed to capture stdout or stderr\n")}

	// Capture the output of the command if ran without errors
	outStr := string(stdout)

	// Split in array by lines
	extList := strings.Split(outStr,"\n")

	fmt.Println("\nGetting latest versions of downloaded extensions\n")
	for _, element := range extList {
		extInfo := strings.Fields(element)

		if len(extInfo) == 2 {
			extId := extInfo[0]
			// Get version number and type
			verFull := extInfo[1]
			verInfo := strings.Split(verFull, "-")
			var verNum, verType string
			if len(verInfo) == 2 {
				// set version number and version qualifier
				verNum, verType = verInfo[0], verInfo[1]
			} else if len(verInfo) == 1 {
				// If version qualifier is missing, it is the latest stable release for that version
				verNum, verType = verInfo[0], "long term (latest stable)"
			}
			
			// if extension is not yet defined
			if _, ok := extMap[extId]; ok {
				// Defined, update if value is greater then one on the map
				if extMap[extId].verNum < verNum {
					fmt.Printf("Updating extension %s to version %s of type %s\n", extId, verNum, verType)
					extMap[extId] = &extVerStruct{verFull, verNum, verType}
				} else if extMap[extId].verNum == verNum && extMap[extId].verType < verType {
					fmt.Printf("Updating extension %s version %s to higher version type %s\n", extId, verNum, verType)
					extMap[extId] = &extVerStruct{verFull, verNum, verType}
				} else {
					fmt.Printf("Skipping extension %s version %s type %s. More recent version is available...\n", extId, verNum, verType)
				}
			} else {
				// Not defined; set it here
				fmt.Printf("Defining extension %s at version %s of type %s\n", extId, verNum, verType)
				extMap[extId] = &extVerStruct{verFull, verNum, verType}
			}
		}
	}
	
	return
}

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

func writeCommonHeader (f *os.File, s string, dbHost string, dbPort string, dbUser string, dbPass string) () {
	
	f.WriteString("<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>\n")
	f.WriteString("<CAST-AutomaticInstall>\n")
	f.WriteString("<!-- Use either ServerName= or ConnectionString= -->\n")
	f.WriteString(fmt.Sprintf(" <ServerInstall ProfileSystem=\"PROFILE_NAME\" ServerType=\"CASTStorageService\" UserSystem=\"%s\" SystemPassword=\"%s\" ServerName=\"%s:%s\" >\n", dbUser, dbPass, dbHost, dbPort))
	f.WriteString(fmt.Sprintf("  <RefreshDatabase DbName=\"%s\" >\n", s))
	f.WriteString("\n")
	f.WriteString("	<!-- Extensions: install most recent that has been downloaded -->\n")
	
	return
}

func writeCommonFooter (f *os.File) () {

 	f.WriteString("\n")
	f.WriteString("   <!-- Extensions: automatically install any required dependencies, using the most recent version that has been downloaded -->\n")
	f.WriteString("   <InstallDependencies strategy=\"TakeLatest\"/>\n")
 	f.WriteString("\n")
	f.WriteString("   <!-- Extensions: prevents installation from the legacy %programdata%\\CAST\\CAST\\<version> location -->\n")
	f.WriteString("   <SkipLookupLegacyUADefaultLocation/>\n")
 	f.WriteString("\n")  
	f.WriteString("  </RefreshDatabase>\n")
	f.WriteString(" </ServerInstall>\n")
	f.WriteString("</CAST-AutomaticInstall>\n")
	
	return
}

	
func main() {
	
	// Make sure required parameters are passed
	if (len(os.Args) != 7) {
		fmt.Printf("Please issue command in the following format: command <AIP install location> <dbHost> <dbPort> <dbUser> <dbPass> <schema prefix>\n")
		fmt.Println("Example: upgradeSchemaExtensions.exe \"C:\\Progra~1\\Cast\\8.2\" localhost 2280 operator CastAIP foo%")
		os.Exit(1)
	}

	aipDir := os.Args[1]
	dbHost := os.Args[2]
	dbPort := os.Args[3]
	dbUser := os.Args[4]
	dbPass := os.Args[5]
	sPrefix := os.Args[6]
	
	// Check if folder location provided is valid
	if _, err := os.Stat(aipDir); os.IsNotExist(err) {
		fmt.Printf("Specified AIP directory location is invalid: %s\n", aipDir)
		fmt.Printf("Please verify and correct\n")
		os.Exit(1)
	} 
	
	// Get current directory
	currDir, panicErr := filepath.Abs(filepath.Dir(os.Args[0]))
    if panicErr != nil {log.Fatal(panicErr)}
	
	// Create temp directory inside current directory
	tempDir, panicErr := ioutil.TempDir(currDir, "CAST")
	if panicErr != nil {log.Fatal(panicErr)}
	//defer os.RemoveAll(tempDir)
	
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s database=postgres sslmode=disable",
		dbHost, dbPort, dbUser, dbPass)
	db, panicErr := sql.Open("postgres", connStr)
	if panicErr != nil {panic(panicErr)}
	defer db.Close()

	fmt.Printf("Enumerating available CAST schemas from %s:%s\n", dbHost, dbPort)
	qry := fmt.Sprintf("select schema_name from information_schema.schemata where schema_name like '%s_mngt' or schema_name like '%s_central' or schema_name like '%s_local' order by schema_name", sPrefix, sPrefix, sPrefix)
	fmt.Printf("Executing query: \n%s\n\n", qry)
	schemaRows, panicErr := db.Query(qry)
	if panicErr != nil {panic(panicErr)}

	// Get a list of latest versions of downloaded extensions
	extMap := make(map[string]*extVerStruct)
	enumInstalledExt(aipDir, extMap)
	
	// Go through list of schemas and run extensions update if necessary
	for schemaRows.Next() {
		var schemaName string
		panicErr = schemaRows.Scan(&schemaName)
		if panicErr != nil {panic(panicErr)}
		fmt.Printf("\nProcessing Schema: %s\n\n", schemaName)

		// Get list of extension for the schema by quering sys_package_version table
		queryStr := fmt.Sprintf("select package_name, version from %s.sys_package_version where package_name like '/com.castsoftware%%' order by package_name", schemaName)
		//fmt.Printf("Executing query: %s\n", queryStr)
		extRows, err1 := db.Query(queryStr)
		
		if err1 != nil {
			fmt.Printf("Error occurred\n%s\n", err1)
		} else {
			// define schema type
			runes := []rune(schemaName)
			schemaType := string(runes[strings.LastIndex(schemaName, "_")+1:])
			fmt.Printf ("Schema %s is of type %s\n", schemaName, schemaType)
			// Create INSTALL_CONFIG_FILE to pass to Server Manager CLI command
			configFileName := fmt.Sprintf("%s\\%s_refresh.xml", tempDir, schemaName)
			logFileName := fmt.Sprintf("%s\\%s_refresh.castlog", tempDir, schemaName)
			f, panicErr := os.Create(configFileName)
			if panicErr != nil {panic(panicErr)}
			defer f.Close()
			
			// Write header for the INSTALL_CONFIG_FILE; vary based on schema type
			if schemaType == "central" {
				writeCommonHeader(f, schemaName, dbHost, dbPort, dbUser, dbPass)
			} else if schemaType == "local" {
				writeCommonHeader(f, schemaName, dbHost, dbPort, dbUser, dbPass)
			} else if schemaType == "mngt" {
				writeCommonHeader(f, schemaName, dbHost, dbPort, dbUser, dbPass)
			} else {
				panic("!!!! Unknown schema type!!!")
			}
		
			var bNeedsUpgrade bool = false
			// Add all extensions to the configuration file
			for extRows.Next() {
				var extID string
				var extVer string
				panicErr = extRows.Scan(&extID, &extVer)
				if panicErr != nil {panic(panicErr)}
				// Remove leading forward slash from extension name
				extID = strings.Replace(extID,"/com.castsoftware", "com.castsoftware", -1)
				
				// Check if extension is downloaded first
				if _, ok := extMap[extID]; ok {
					// check if version of downloaded extension is greater then what is installed in schema
					if extMap[extID].verFull != extVer {
						fmt.Printf("Extension '%s': installed: %s; available: %s. Marking for upgrade!!!\n", extID, extVer, extMap[extID].verFull)
						bNeedsUpgrade = true // Marking schema for upgrade
					} else {
						fmt.Printf("Extension '%s': installed/available: %s. No upgrade required.\n", extID, extVer)
					}
				} else {
					fmt.Printf("Extension %s is not yet downloaded. Please download first before running update.\n")
				}
				// Add extension entry to INSTALL_CONFIG_FILE
				f.WriteString(fmt.Sprintf("   <Plugin id=\"%s\"/>\n", extID))
			}
			// Write footer for the INSTALL_CONFIG_FILE
			writeCommonFooter(f)
			f.Sync()
			
			// if schema was marked for upgrade, execute server manager Refresh
			if bNeedsUpgrade {
				fmt.Printf("Schema %s marked for upgrade. Kicking off process.. \n", schemaName)
				
				// Execute CAST AIP Server manager to refresh schema
				cmd := exec.Command("cmd", "/c", aipDir+"\\servman.exe", "-INSTALL_CONFIG_FILE", "('"+configFileName+"')", "-LOG", "('"+logFileName+"')")
				fmt.Printf("Executing command: %s\n", cmd.Args)
				
				if panicErr := cmd.Run(); panicErr != nil {
					log.Fatalf("cmd.Run() failed with %s\n", panicErr)
				}
				bNeedsUpgrade = false // reset upgrade flag for next schema
			} else {
				fmt.Printf("Schema %s does not require an upgrade. Skipping.. \n", schemaName)
			}
		}
	}
}