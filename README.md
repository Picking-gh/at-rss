# aria2c-rss

This is a modified version for aria2c inspired by transmission-rss. It allows users to extract torrents filtered by specific keywords from subscribed RSS feeds and initiate downloads using aria2c's RPC. The transmission-rss mentioned here is my customized version based on trishika/transmission-rss. Special thanks to trishika/transmission-rss for the foundational framework.

Note: When "trick" is enabled, the hash of each seed from the item element is extracted using a user specified regular expression pattern that is reconstructed as a magnet link instead of the link in the enclosure element. This approach may not be universally applicable; one is encouraged to modify the source code as needed. 

Please refer to aria2c-rss.conf for additional keyword filtering details.

THIS SCRIPT IS SPECIFICALLY CONFIGURED FOR BT TORRENTS RSS FEEDS.