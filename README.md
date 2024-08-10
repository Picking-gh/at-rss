# at-rss: A RSS Feed Parser for aria2c and transmission

The origin of this project is that I couldn't compile FlexGet for my old ARMv7 development board, so I decided to create a simple replacement myself.

This project, named **at-rss** (covering both **aria2c-rss** and **transmission-rss**), is rebuilt from [transmission-rss](https://github.com/trishika/transmission-rss). It enables users to filter torrents from subscribed RSS feeds using specific keywords and initiate downloads via the RPC server of either aria2c or transmission. Special thanks to trishika for the initial version.

## Key Features

- **Magnet Link Extraction:**  
  When an `extracter` is specified in the `at-rss.conf` file, the script extracts a hash from a designated element (e.g., `title`, `link`, `description`, `enclosure`, or `guid`) using a user-defined regular expression pattern. This hash is then used to reconstruct a magnet link, replacing the original link in the `enclosure` element. This script uses `bith`, the pattern is usually `"([a-f0-9]{40})"`.

- **Support for aria2c and transmission:**  
  This script supports both aria2c and Transmission, giving users the flexibility to choose their preferred torrent client—whether it's for downloading BT or PT, for example. This is done by specifying `aria2c` or `transmission` for each feed. This is probably not the best way, but it works. 

- **Keyword Filtering:**  
  Filters (not using regular expressions) can be applied to the `title` element of RSS items, allowing for both inclusion and exclusion criteria. These filters support simple combining conditions with AND/OR logic. For more details on keyword filtering and configuration, please refer to the `at-rss.conf` file..

**Note:**  
The magnet link extraction feature may not be universally applicable; one is encouraged to modify the source code as needed.

**IMPORTANT:**  
This script is specifically configured for RSS feeds containing BitTorrent torrents.
