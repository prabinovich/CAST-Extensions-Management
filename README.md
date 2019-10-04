# CAST Extensions Management Tools
The tools are built on top of Server Manager CLI to simplify management of extensions on a single or multiple applications at the same time. The project includes the following Go scripts:

1. downloadExtensions.go - script that identifies available extensions and installs or upgrades them based on the provided command line arguments
2. UpgradeSchemaExtensions.go - script that will evaluate extensions installed in designated CAST schemas and see if any of those extensions have newer versions downloaded. If there are, it will update those schemas with latest versions of those extensions
3. installSchemaExtensions.go - script is used to install or remove specific extensions from a single or multiple schemas

The project includes both the scripts and executables. To run the executables you don't need anything other then EXE file itself. If you need to make changes/updates to the scripts, you can do that and use the following instructions to rebuild:

To rebuild one of the build programs:
- Download and install Go: https://golang.org/dl/
- Use this command to rebuild the file:

cd into BIN directory of the project where EXE binaries are located (.\bin)
Compile using this command line: 

go build -o downloadExtensions.exe ..\src\downloadExtensions\downloadExtensions.go
-or-
go build -o upgradeSchemaExtensions_AIP837.exe ..\src\upgradeSchemaExtensions\upgradeSchemaExtensions_AIP837.go

DownloadExtensions Script
==================================
The script will need to be provided the following parameters:
- AIP install location - location of where CAST AIP is installed
- Extend Server URL - Extend server URL (ex. https://extend.castsoftware.com:443/V2/api/v2)
- user - username for Extend website
- password - password for Extend website
- upgrade|install - upgrade will only identify new versions of already downloaded extension. Install will download available extensions that have not yet been installed (aka. run both in order to get latest of installed and newly available extensions)
- official|all - argument indiciates whether to download CAST Labs and CAST Communition extensions (all) or to consider only extensions published by CAST Product (official)
- stable|latest - indicates whether to download only stable (skip Alpha and Beta releases) or the latest available
- list|download - determines whether to provide the list of extensions to install/update to or to actually download them
  
Here's an example of how you can execute the program:
downloadExtensions.exe "C:\Program Files\Cast\8.3" "https://extend.castsoftware.com:443/V2/api/v2" p.rabinovich@castsoftware.com xxxxxx upgrade all stable download

Note: There is a problem with the ExtensionDownloader.exe CLI in CAST AIP < 8.3.5 that fails to authenticate against when trying to download extensions from https://extend.castsoftware.com:443/V2/api/v2 . The issue is resolved as part of 8.3.5. If using earlier veresion, as workaround, please use https://extend.castsoftware.com:443/product instead when executing the script. 

UpgradeSchemaExtensions Script
===============================
The script will need to be provided the following parameters:
- AIP install location - location of where CAST AIP is installed; Make sure to user short folder notation (i.e. C:\Progra~1\... instead of C:\Program Files\...)
- db host - name or IP of the server that hosts cast storage services CSS
- db port - port number on which CSS runs
- schema prefix - schema prefix that should be considered for upgrade. The prefix can include % as part of the string to represent a wildcard or just % to include all schemas hosted on the designed CSS server

Here's an example of how to execute the program:
upgradeSchemaExtensions.exe "C:\Program Files\Cast\8.3.3" localhost 2282 foo%

InstallSchemaExtensions Script
===============================
Script is used to install or remove specific set of extensions from a single or multiple schemas. The script will need to be provided the following parameters when executed:
- AIP install location - location of where CAST AIP is installed; Make sure to user short folder notation (i.e. C:\Progra~1\... instead of C:\Program Files\...). Otherwise put the path in quotes.
- db host - name or IP of the server that hosts cast storage services CSS (ex: localhost)
- db port - port number on which CSS runs (ex: 2282)
- schema prefix - schema prefix that should be considered for upgrade. The prefix can include % as part of the string to represent a wildcard or just % to include all schemas hosted on the designed CSS server. For instance: foo%
- Config File Path - location of the file that defines which extensions to install or remove from the designated schemas. The file should list one extension name per line. If you want to install specific version of extension, add equals sign "=" followed by the the extension version number. For instance: com.castsoftware.qualitystandards=20190923.0.0-funcrel. To remove an extension, use "remove" in place of the version number, such as: com.castsoftware.qualitystandards=remove
- Info or Update - flag that indicates whether to update schemas with requested extensions or to report which extensions will be installed in which schemas

Here's an example of how to execute the program:
installSchemaExtensions.exe "C:\Program Files\Cast\8.3" localhost 2282 [a-z].* "c:\temp\extensions.txt" update

