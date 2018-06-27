# ExtensionsUpgrade
Tools to download latest versions of CAST AIP extensions and upgrade respective schemas. The project currently includes two Go scripts:

1. downloadUpgradedExtensions.go - script that goes through list of installed extensions, identifies available upgrades and downloads more recent versions that are available
2. UpgradeSchemaExtensions.go - script that will evaluate extensions installed in designated CAST schemas and see if any of those extensions have newer versions downloaded. If there are, it will update those schemas with latest versions of those extensions

The project includes both the scripts and executables. To run the executables you don't need anything other then EXE file itself. If you need to make changes/updates to the scripts, you can do that and use the following instructions to rebuild:

To rebuild one of the build programs:
- Download and install Go: https://golang.org/dl/
- Use this command to rebuild the file:

go build -o test1.exe test1.go

DownloadUpgradedExtensions Script
==================================
The script will need to be provided the following parameters:
- AIP install location - location of where CAST AIP is installed
- Extend Server URL - Extend server URL (ex. https://extend.castsoftware.com:443/V2/api/v2)
- user - username for Extend website
- password - password for Extend website
- stableOnly or latest - determines wether to download only stable (skip Alpha and Beta releases) or the latest available
  
Here's an example of how you can execute the program:
downloadUpgradedExtensions.exe "C:\Program Files\Cast\8.3.3" https://extend.castsoftware.com:443/V2/api/v2 p.rabinovich@castsoftware.com xxxxxx stableOnly

Note: Currently there is a problem with the ExtensionDownloader.exe CLI that fails to authenticate against when trying to download extensions from https://extend.castsoftware.com:443/V2/api/v2 . The ticket was raised for this issue and the latest status is that this is a known issue that planned for resolution as part of 8.3.5 (to be released in July). Meanwhile, as a workaround, please use https://extend.castsoftware.com:443/product instead when executing the script. 

UpgradeSchemaExtensions Script
===============================
The script will need to be provided the following parameters:
- AIP install location - location of where CAST AIP is installed
- db host - name or IP of the server that hosts cast storage services CSS
- db port - port number on which CSS runs
- schema prefix - schema prefix that should be considered for upgrade. The prefix can include % as part of the string to represent a wildcard or just % to include all schemas hosted on the designed CSS server

Here's an example of how to execute the program:
upgradeSchemaExtensions.exe "C:\Program Files\Cast\8.3.3" localhost 2282 foo%
