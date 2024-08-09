# at-rss

This modified version of the script, referred to as at-rss (encompassing both aria2c-rss and transmission-rss), is inspired by transmission-rss. It enables users to extract torrents filtered by specific keywords from subscribed RSS feeds and initiate downloads using the RPC of either aria2c or transmission. The transmission-rss mentioned here is a customized version based on trishika/transmission-rss. Special thanks to trishika/transmission-rss for providing the foundational framework.

Note: When an extractor is specified in the at-rss.conf file, the hash of each item is extracted from the element using a user-defined regular expression pattern. This hash is then used to reconstruct a magnet link instead of using the link in the enclosure element. This approach may not be universally applicable, so you may need to modify the source code as needed.

For additional details on keyword filtering, please refer to the at-rss.conf file.

IMPORTANT: This script is specifically configured for RSS feeds containing BitTorrent torrents.