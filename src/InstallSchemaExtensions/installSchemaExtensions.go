 // Use for CAST AIP v8.3.x
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
	"bufio"
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

// Get list of extensions currently installed on schema
func enumSchemaExt (db *sql.DB, schemaName string, schemaExtMap map[string]string) () {
	// Get list of extension for the schema by quering sys_package_version table
	queryStr := fmt.Sprintf("select package_name, version from %s.sys_package_version where package_name like '/com.castsoftware%%' order by package_name", schemaName)
	//fmt.Printf("Executing query: %s\n", queryStr)
	extRows, err1 := db.Query(queryStr)
	
	if err1 != nil {
		fmt.Printf("Error occurred while enumerating schema packages\n%s\n", err1)
	} else {
		fmt.Printf("Reading extensions installed on schema %s\n", schemaName)
		for extRows.Next() {
			var extID string
			var extVer string
			panicErr := extRows.Scan(&extID, &extVer)
			if panicErr != nil {panic(panicErr)}
			// Remove leading forward slash from extension name
			extID = strings.Replace(extID,"/com.castsoftware", "com.castsoftware", -1)
			schemaExtMap[extID] = extVer
			fmt.Printf("%s=%s\n", extID, extVer)
		}
	}
}

func readExtConfigs (aipDir string, extFilePath string, aExt2Install *[][]string) {
	
	//extMap map[string]*extVerStruct
	extMap := make(map[string]*extVerStruct)
	
	fmt.Printf("\nEnumerating available extensions...\n\n")
	cmd := exec.Command(aipDir + "\\ExtensionDownloader.exe", "list", "installed")
	//fmt.Printf("Executing command: %s\n", cmd.Args)
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

	fmt.Println("\nGetting latest versions of available extensions\n")
	for _, element := range extList {
		aExt2Install := strings.Fields(element)

		if len(aExt2Install) == 2 {
			extId := aExt2Install[0]
			// Get version number and type
			verFull := aExt2Install[1]
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
	
	// Read-in the extensions configuration file into an array
	fmt.Printf("Read-in the extensions configuration file\n")
	//aExt2Install := [][]string{}
	f, err1 := os.Open(extFilePath)
	if err1 != nil {
			fmt.Printf("\nError occurred while opening Extenions configuration file. Error reported: %s\n", err1)
	} else {
		fmt.Printf("The following extensions were found in the configuration file:\n")
	    scanner := bufio.NewScanner(f)
	    count := 1
	    for scanner.Scan() {
	        line := scanner.Text()
	        columns := strings.Split(line, "=")
	        if _, ok := extMap[columns[0]]; ok {
		        row := []string{}
		        if len(columns) < 2 {
		        	// Use the latest version if not specified
		        	row = []string{columns[0],extMap[columns[0]].verFull}
		        } else {
		        	// Use the version provided in the config file
		        	row = []string{columns[0],columns[1]}
		        }
		        *aExt2Install = append(*aExt2Install, row)
		        fmt.Printf("%d: %s %s\n", count, row[0], row[1])
		        count++
	        } else {
	        	fmt.Printf("Extension %s referenced in the configuration file is not found.\n", columns[0])
	        }
	    }
	}
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
	f.WriteString(fmt.Sprintf("  <ManagePlugins SchemaPrefix=\"%s\" >\n", s))
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
	f.WriteString("  </ManagePlugins>\n")
	f.WriteString(" </ServerInstall>\n")
	f.WriteString("</CAST-AutomaticInstall>\n")
	
	return
}

	
func main() {
	
	// Make sure required parameters are passed
	if (len(os.Args) != 8) {
		fmt.Printf("Please issue command in the following format: command <AIP-install-location> <dbHost> <dbPort> <dbUser> <dbPass> <schema prefix> <Ext-config-file-path>\n")
		fmt.Println("Example: InstallSchemaExtensions.exe \"C:\\Program Files\\Cast\\8.2\" localhost 2280 operator CastAIP foo% c:\\temp\\extensions.txt")
		os.Exit(1)
	}

	aipDir := os.Args[1]
	dbHost := os.Args[2]
	dbPort := os.Args[3]
	dbUser := os.Args[4]
	dbPass := os.Args[5]
	sPrefix := os.Args[6]
	extFilePath := os.Args[7]
	
	// Check if folder location provided is valid
	if _, err := os.Stat(aipDir); os.IsNotExist(err) {
		fmt.Printf("Specified AIP directory location is invalid: %s\n", aipDir)
		fmt.Printf("Please verify and correct\n")
		os.Exit(1)
	}
	
	// Check if provided extensions configuration file valid
	if _, err := os.Stat(extFilePath); os.IsNotExist(err) {
		fmt.Printf("Specified extensions configuraiton file is invalid: %s\n", extFilePath)
		fmt.Printf("Please verify and correct\n")
		os.Exit(1)
	}
	
	aExt2Install := [][]string{}
	readExtConfigs(aipDir, extFilePath, &aExt2Install)
	
	// Get current directory
	currDir, panicErr := filepath.Abs(filepath.Dir(os.Args[0]))
    if panicErr != nil {log.Fatal(panicErr)}
	
	// Create temp directory inside current directory
	tempDir, panicErr := ioutil.TempDir(currDir, "CAST")
	if panicErr != nil {log.Fatal(panicErr)}
	// Clean up temporary directory once process terminates
	//defer os.RemoveAll(tempDir)
	
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s database=postgres sslmode=disable",
		dbHost, dbPort, dbUser, dbPass)
	db, panicErr := sql.Open("postgres", connStr)
	if panicErr != nil {panic(panicErr)}
	defer db.Close()

	// Find all CAST management schemas using the naming convention
	fmt.Printf("Enumerating available CAST management schemas on %s:%s\n", dbHost, dbPort)
	qry := fmt.Sprintf("select schema_name from information_schema.schemata where schema_name like '%s_mngt' order by schema_name", sPrefix)
	//fmt.Printf("Executing query: \n%s\n\n", qry)
	schemaRows, panicErr := db.Query(qry)
	if panicErr != nil {panic(panicErr)}
	
	// Go through list of schemas and run extensions update if necessary
	for schemaRows.Next() {
		var schemaName string
		panicErr = schemaRows.Scan(&schemaName)
		if panicErr != nil {panic(panicErr)}
		// Get type and prefix for the schema
		runes := []rune(schemaName)
		//schemaType := string(runes[strings.LastIndex(schemaName, "_")+1:])
		schemaPrefix := string(runes[:strings.LastIndex(schemaName, "_")])
		fmt.Printf("Processing Schema: %s\n", schemaName)

		// Get a list of extensions already installed on schema
		schemaExtMap := make(map[string]string)
		enumSchemaExt(db, schemaName, schemaExtMap)

		// Create INSTALL_CONFIG_FILE to pass to Server Manager CLI command
		configFileName := fmt.Sprintf("%s\\%s_refresh.xml", tempDir, schemaName)
		logFileName := fmt.Sprintf("%s\\%s_refresh.castlog", tempDir, schemaName)
		f, panicErr := os.Create(configFileName)
		if panicErr != nil {panic(panicErr)}
		defer f.Close()
		
		// Write header for the INSTALL_CONFIG_FILE
		writeCommonHeader(f, schemaPrefix, dbHost, dbPort, dbUser, dbPass)
	
		// Add all extensions to the configuration file
		fmt.Printf("Number of extenions updates to %s schema: %d\n", schemaName, len(aExt2Install))
		var bInstallFlag bool = false
		for _, element := range aExt2Install {
			if element[1] == "remove" { // Handle special use case to remove extension
				// Check if extension exists
				if _, ok := schemaExtMap[element[0]]; ok {
				    fmt.Printf("Extension %s found on schema %s. Marking for removal\n", element[0], schemaName)
				    f.WriteString(fmt.Sprintf("   <Plugin id=\"%s\" version=\"%s\"/>\n", element[0], element[1]))
				    bInstallFlag = true // Marking schema for update
				} else {
					fmt.Printf("Extension %s was NOT found on schema %s... skipping\n", element[0], schemaName)
				}
			} else {
				// Check if the extension is already installed and the version matches up
				if schemaExtMap[element[0]] == element[1] {
					fmt.Printf("%s=%s extension is already installed on %s schema... skipping\n", element[0], element[1], schemaName)
				} else {
					// Add extension entry to INSTALL_CONFIG_FILE
					fmt.Printf("Addiing %s=%s extension to be installed on %s schema\n", element[0], element[1], schemaName)
					f.WriteString(fmt.Sprintf("   <Plugin id=\"%s\" version=\"%s\"/>\n", element[0], element[1]))
					bInstallFlag = true // Marking schema for update
				}
			}
		}
		
		// Write footer for the INSTALL_CONFIG_FILE
		writeCommonFooter(f)
		f.Sync()

		if bInstallFlag {
			fmt.Printf("Kicking off schema update process for %s\n", schemaName)
			
			// Execute CAST AIP Server manager to refresh schema
			cmd := exec.Command("cmd", "/c", aipDir+"\\servman.exe", "-INSTALL_CONFIG_FILE", "('"+configFileName+"')", "-LOG", "('"+logFileName+"')")
			fmt.Printf("Executing command: %s\n", cmd.Args)
			fmt.Printf("Running... \n")
			
			if panicErr := cmd.Run(); panicErr != nil {
				//log.Fatalf("cmd.Run() failed with %s\n", panicErr)
				fmt.Printf("cmd.Run() failed with %s\n", panicErr)
			} else {
				fmt.Printf("Schema %s processed successfully\n", schemaName)
			}
		} else {
			fmt.Printf("No extensions found to install or remove. Skipping schema %s\n", schemaName)
		}
		bInstallFlag = false // reset upgrade flag for next schema
	}
	fmt.Printf("\nExtensions update process completed!!!\n\n")
}
