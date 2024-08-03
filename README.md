# aria2c-rss
A version of aria2c modeled after transmission-rss, used to extract torrents with specified keywords from subscribed rss feeds, and add downloads through aria2c's rpc. The transmissiont-rss mentioned here is my modified version from trishika/transmission-rss. Thanks to trishika/transmission-rss for providing the infrastructure.

Note: if trick is set to true, the hash of the seed is extracted from each item in the feed according to the regular expression pattern to reconstruct the magnetic link instead of directly downloading the link in the enclosure. It cannot be guaranteed to be universal, please modify the source code if necessary. Refer to aria2c-rss.conf for more information.

THIS IS ONLY ADOPTED FOR BT TORRENTS RSS FEEDS.