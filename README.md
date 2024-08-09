# at-rss: A Modified RSS Feed Parser for aria2c and transmission

This modified version of the script, referred to as **at-rss** (encompassing both **aria2c-rss** and **transmission-rss**), is based on transmission-rss. It allows users to extract torrents filtered by specific keywords from subscribed RSS feeds and initiate downloads using the RPC of either aria2c or transmission. The transmission-rss mentioned here is a customized version based on trishika/transmission-rss. Special thanks to trishika/transmission-rss for providing the foundational framework.

## Key Features

- **RSS Feed Parsing:**  
  The script is designed to parse RSS feeds containing BitTorrent torrents and identify items matching user-defined filters.

- **Keyword Filtering:**  
  Filters can be applied to the `title` element of RSS items, allowing for both inclusion and exclusion criteria. The filters support complex conditions, such as multiple keywords with AND/OR logic.

- **Magnet Link Extraction:**  
  When an extractor is specified in the `at-rss.conf` file, the script can extract a hash from a specified element (e.g., `title`, `link`, `description`, `enclosure`, or `guid`) using a user-defined regular expression pattern. This hash is then used to reconstruct a magnet link, replacing the link in the `enclosure` element.

- **Flexible Configuration:**  
  The configuration file (`at-rss.conf`) allows users to specify RPC server names, feed URLs, filters, extractors, and intervals. The script automatically falls back to default settings if a provided RPC server name is not fully specified.

- **Support for aria2c and transmission:**  
  The script supports both aria2c and transmission, enabling users to choose their preferred torrent client. If only the name of an RPC server is provided, it defaults to a standard value for dialing out.

**Note:**  
The magnet link extraction feature may not be universally applicable; one is encouraged to modify the source code as needed.

For additional details on keyword filtering and configuration, please refer to the `at-rss.conf` file.

**IMPORTANT:**  
This script is specifically configured for RSS feeds containing BitTorrent torrents.
