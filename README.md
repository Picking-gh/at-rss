# at-rss: A RSS Feed Parser for aria2c and transmission

The origin of this project is that I couldn't compile FlexGet for my old ARMv7 development board, so I decided to create a simple replacement myself.

This project, named **at-rss** (covering both **aria2c-rss** and **transmission-rss**), is rebuilt from [transmission-rss](https://github.com/trishika/transmission-rss). It enables users to filter torrents from subscribed RSS feeds using specific keywords and initiate downloads via the RPC server of either aria2c or transmission. Special thanks to trishika for the initial version.

## Key Features

- **Magnet Link Extraction:**  
  When an `extracter` is specified in the `at-rss.conf` file, the script extracts a hash from a designated element (e.g., `title`, `link`, `description`, `enclosure`, or `guid`) using a user-defined regular expression pattern (this script use `bith`, the pattern is `"([a-f0-9]{40})"`). This hash is then used to reconstruct a magnet link, replacing the original link in the `enclosure` element.

- **Support for aria2c and transmission:**  
  The script supports both aria2c and Transmission, allowing users to choose their preferred torrent client—one for downloading BT and the other for PT. You can simply specify either `aria2c` or `transmission` to go, as many users start them with the default configuration.

- **Keyword Filtering:**  
  Filters can be applied to the `title` element of RSS items, allowing for both inclusion and exclusion criteria. The filters support complex conditions, such as multiple keywords with AND/OR logic. For additional details on keyword filtering and configuration, please refer to the `at-rss.conf` file.

**Note:**  
The magnet link extraction feature may not be universally applicable; one is encouraged to modify the source code as needed.

**IMPORTANT:**  
This script is specifically configured for RSS feeds containing BitTorrent torrents.
