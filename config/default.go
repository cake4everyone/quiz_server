package config

const DEFAULTCONFIG = `# Quiz4Everyone - Server
# This Project was created by Kesuaheli for Cake4Everyone

# This is the main configuration file for the server backend of Quiz4Everyone. You can change any
# value as long as it matches the corresponding syntax. Changes at your own risk, I have warned you!
#
# With that out of the way... You can always get the default config by DELETING or RENAMING this
# file. (Or anything else you want to do with it, as long as there is no file called "config.yaml"
# in the same folder)

google:
  # API Key for Google to acces the Google Sheets API
  api_key: YOUR_KEY_HERE
  # The ID of your Google Spreadsheet
  # (You find it in the URL for example)
  spreadsheetID:

webserver:
  # The port to start the webserver on.
  port: 51445
  # password is the very first thing a client must send in order to verify their connection. Must
  # not be an empty string.
  password: PleaseChange123
`
