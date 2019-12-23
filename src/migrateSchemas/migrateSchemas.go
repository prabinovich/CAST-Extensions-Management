 // For use with CAST AIP v8.3.x and above
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
	"errors"
	"bytes"
)

// Get the version of the schame being processed
func checkSchemaVer(db *sql.DB, schemaName string) (string, error) {
	// Get schema version
	var schemaVer string
	queryStr := fmt.Sprintf("select version from %s.sys_package_version where package_name = 'CORE_PMC'", schemaName)
	err1 := db.QueryRow(queryStr).Scan(&schemaVer)
	
	if err1 != nil {
		log.Printf("Error occurred while checking schema version\n%s\n", err1)
		return "", err1
	} else {
		return schemaVer, nil
	}
}

// Read in configuration file and confirm that all parameters have been properly set
func readConfigFile (cfgFilePath string) (map[string]string, error) {
	
	configParams := make(map[string]string)
	
	log.Printf("Reading configuration parameters in file '%s'...\n", cfgFilePath)
	f, err1 := os.Open(cfgFilePath)
	if err1 != nil {
			log.Printf("\nError occurred while opening configuration file: %s\n", err1)
			return nil, err1
	} else {
		log.Printf("The following extensions were found in the configuration file:\n")
	    scanner := bufio.NewScanner(f)
	    //count := 1
	    // Read in all configuration parameters into a map
	    for scanner.Scan() {
	        line := scanner.Text()
	        columns := strings.Split(line, "=")
	        if (len(columns) != 2) {
	        	log.Printf("Incorrect format for the configuration file. Error in line:\n")
	        	log.Printf("---> %s\n", line)
	        	return nil, errors.New("Incorrect format for the configuration file.")
	        } else {
	        	log.Printf("Found configuration key/pair value: %s=%s\n", columns[0], columns[1])
	        	configParams[columns[0]] = columns[1]
	        }
	    }
	    
	    // Check to make sure all parameters that we are expecting were included in the configuration file
	    if _, ok := configParams["AIP_HOME"]; !ok {log.Printf("Parameter 'AIP_HOME' not found in the configuration file. Please fix."); return nil, errors.New("Parameter undefined")}
	    if _, ok := configParams["AIP_VERSION"]; !ok {log.Printf("Parameter 'AIP_VERSION' not found in the configuration file. Please fix."); return nil, errors.New("Parameter undefined")}
	    if _, ok := configParams["CAST_DEFAULT_DELIVERY_DIR"]; !ok {log.Printf("Parameter 'CAST_DEFAULT_DELIVERY_DIR' not found in the configuration file. Please fix."); return nil, errors.New("Parameter undefined")}
	    if _, ok := configParams["CAST_DEFAULT_DEPLOY_DIR"]; !ok {log.Printf ("Parameter 'CAST_DEFAULT_DEPLOY_DIR' not found in the configuration file. Please fix."); return nil, errors.New("Parameter undefined")}
	    if _, ok := configParams["CAST_DEFAULT_LISA_DIR"]; !ok {log.Printf("Parameter 'CAST_DEFAULT_LISA_DIR' not found in the configuration file. Please fix."); return nil, errors.New("Parameter undefined")}
	    if _, ok := configParams["CAST_LOG_ROOT_PATH"]; !ok {log.Printf("Parameter 'CAST_LOG_ROOT_PATH' not found in the configuration file. Please fix."); return nil, errors.New("Parameter undefined")}
	    
	    // Check if folder location provided is valid
		if _, err1 := os.Stat(configParams["AIP_HOME"]); os.IsNotExist(err1) {
			log.Printf("Specified AIP directory location is invalid: %s\n", configParams["AIP_HOME"])
			log.Printf("Please verify and correct\n")
			return nil, err1
		}
	}
	
	return configParams, nil
}

// Update CMS prefernces for the specified schema
func udpateDeliveryPath(db *sql.DB, schemaName string, configParams map[string]string) (error) {
	// Define SQL statement to update schema
	sqlStr := fmt.Sprintf("UPDATE %s.cms_pref_sources SET serverpath=$1;", schemaName)
	
	_, err1 := db.Exec(sqlStr, configParams["CAST_DEFAULT_DELIVERY_DIR"])
	
	if err1 != nil {
		log.Printf("Error occurred while updating db\n%s\n", err1)
		return err1
	} else {
		return nil
	}
}

// Update CMS prefernces for the specified schema
func udpateSchemaCmsPrefs(db *sql.DB, schemaName string, configParams map[string]string) (error) {
	// Define SQL statement to update schema
	sqlStr := fmt.Sprintf("UPDATE %s.cms_pref_sources SET serverpath=$1, deploypath=$2, logrootpath=$3, workingpath=$4, temporarypath=$4;", schemaName)
	
	_, err1 := db.Exec(sqlStr, configParams["CAST_DEFAULT_DELIVERY_DIR"], configParams["CAST_DEFAULT_DEPLOY_DIR"], 
		configParams["CAST_LOG_ROOT_PATH"], configParams["CAST_DEFAULT_LISA_DIR"])
	
	if err1 != nil {
		log.Printf("Error occurred while updating db\n%s\n", err1)
		return err1
	} else {
		return nil
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

func main() {
	
	// Set schema counters
	identifiedSchemas, processedSchemas := 0, 0
	
	// Make sure required parameters are passed
	if (len(os.Args) !=8) {
		log.Printf("Please issue command in the following format: command <configFile> <dbHost> <dbPort> <dbUser> <dbPass> <schema regex prefix> <info|update>\n")
		fmt.Println("Example: migrateSchemas.exe \"c:\\temp\\config.txt\" localhost 2282 operator CastAIP [a-z].* update")
		os.Exit(1)
	}

	cfgFilePath := os.Args[1]
	dbHost := os.Args[2]
	dbPort := os.Args[3]
	dbUser := os.Args[4]
	dbPass := os.Args[5]
	sPrefix := os.Args[6]
	
	// info|update
	infoOrUpdate := os.Args[7]
	if !strings.EqualFold(infoOrUpdate, "info") && !strings.EqualFold(infoOrUpdate, "update") {
		fmt.Println("Incorrect parameter value specified. Please provide one of the following options: info|update\n")
		os.Exit(1)
	}
	
	// Read in additional configuarations file the file
	configFileParams, err1 := readConfigFile(cfgFilePath)
	if err1 != nil {log.Printf("Error occured... exiting: %s\n", err1); os.Exit(1)}
	
	// Get current directory
	currDir, err1 := filepath.Abs(filepath.Dir(os.Args[0]))
    if err1 != nil {log.Printf("Error occured... exiting: %s\n", err1); os.Exit(1)}
	
	// Create temp directory inside current directory
	tempDir, err1 := ioutil.TempDir(currDir, "CAST")
	if err1 != nil {log.Printf("Error occured... exiting: %s\n", err1); os.Exit(1)}
	// Clean up temporary directory once process terminates
	//defer os.RemoveAll(tempDir)
	
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s database=postgres sslmode=disable",
		dbHost, dbPort, dbUser, dbPass)
	db, err1 := sql.Open("postgres", connStr)
	if err1 != nil {log.Printf("Error occured... exiting: %s\n", err1); os.Exit(1)}
	defer db.Close()

	// Find all CAST management schemas using the naming convention
	log.Printf("Enumerating available CAST management schemas on %s:%s\n", dbHost, dbPort)
	//qry := fmt.Sprintf("select schema_name from information_schema.schemata where schema_name like '%s_mngt' order by schema_name", sPrefix)
	qry := fmt.Sprintf("select count(*) from information_schema.schemata where schema_name ~ '^%s_mngt$'", sPrefix)
	row := db.QueryRow(qry)
	err1 = row.Scan(&identifiedSchemas)
	if err1 != nil {log.Printf("Error occured... exiting: %s\n", err1); os.Exit(1)}
	
	qry = fmt.Sprintf("select schema_name from information_schema.schemata where schema_name ~ '^%s_mngt$' order by schema_name", sPrefix)
	//log.Printf("Executing query: \n%s\n\n", qry)
	schemaRows, err1 := db.Query(qry)
	if err1 != nil {log.Printf("Error occured... exiting: %s\n", err1); goto Exit}
	log.Printf("Identified %d applications that potentialy require a migration\n", identifiedSchemas)
	
	// Go through list of schemas and run extensions update if necessary
	for schemaRows.Next() {
		var schemaName string
		err1 := schemaRows.Scan(&schemaName)
		if err1 != nil {log.Printf("Error occured... exiting: %s\n", err1); goto Exit}
		// Get type and prefix for the schema
		//runes := []rune(schemaName)
		//schemaType := string(runes[strings.LastIndex(schemaName, "_")+1:])
		//schemaPrefix := string(runes[:strings.LastIndex(schemaName, "_")])
		log.Printf("Processing Schema: %s\n", schemaName)
		logFileName := fmt.Sprintf("%s\\%s_migrate.castlog", tempDir, schemaName)

		// Check to see if version of schema is older then version of CAST AIP and it needs to be migrated
		schemaVer, err1 := checkSchemaVer(db, schemaName)
		if err1 != nil {
			log.Printf("Error getting schema '%s' version: %s\n", schemaName, err1)
		}
		var bInstallFlag bool = false
		if (schemaVer < configFileParams["AIP_VERSION"]) {
			log.Printf("Schema '%s' is version '%s'. Marking for upgrade to version '%s'\n", schemaName, schemaVer, configFileParams["AIP_VERSION"])
			bInstallFlag = true
		} else {
			log.Printf("Schema '%s' is version '%s' and it does not require an upgrade\n", schemaName, schemaVer)
			bInstallFlag = false
		}

		// If upgrade flag is set, migrate the schema
		if bInstallFlag {
			
			// Run command line if instructed
			if infoOrUpdate == "update" {
				log.Printf("Kicking off schema update process for '%s' schema\n", schemaName)
				
				// Update CMS preferneces in MNGT schema
				log.Printf("Updating CMS preferences for schema '%s'\n", schemaName)
				err1 := udpateDeliveryPath(db, schemaName, configFileParams)
				if err1 != nil {
					log.Printf("Failed to update Delivery location preferences for schema %s\n", schemaName)
					log.Printf("Error: %s\n", err1)
					log.Printf("Skipping migration for current schema\n")
				} else {
					// Execute CAST AIP Server manager to refresh schema
					cmd := exec.Command("cmd", "/c", configFileParams["AIP_HOME"]+"\\servman.exe", 
					"-MODIFY_COMBINED(management:="+schemaName+", assessmentModel:=replaceByNewAssessmentModel)", 
					"-CONNECT_STRING('"+dbHost+":"+dbPort+"',"+dbUser+","+dbPass+")", 
					"-LOG('"+logFileName+"', -IMMEDIATE)")
					// log.Printf("Command to execute: %s\n", cmd.Args)
					log.Printf("Running migration command... \n")
					var stderr bytes.Buffer
					cmd.Stderr = &stderr
					if err1 := cmd.Run(); err1 != nil {
						log.Printf("cmd.Run() failed with %s\n", err1)
						log.Printf(stderr.String())
					} else {
						log.Printf("Schema %s processed successfully\n", schemaName)
						processedSchemas++
						err1 := udpateSchemaCmsPrefs(db, schemaName, configFileParams)
						if err1 != nil {
							log.Printf("Failed to update CMS schema preferences for schema %s\n", schemaName)
							log.Printf("Error: %s\n", err1)
						}
					}
				}
			} else {
				log.Printf("Running command in INFO mode. Schema %s will be updated if command is run in UPDATE mode.\n", schemaName)
			}
		} else {
			log.Printf("Schema '%s' did not not need to be migrated...skipping\n", schemaName)
		}
		bInstallFlag = false // reset upgrade flag for next schema
	}
	log.Printf("Migration successfully completed for %d schemas!!!\n\n", processedSchemas)
	
Exit:
	log.Printf("Exiting....\n")
}
