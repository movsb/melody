#!/bin/bash

set -eu

yt-dlp \
	--add-metadata \
	--embed-thumbnail \
	--embed-subs \
	--no-playlist \
	--force-ipv4 \
	--proxy socks5://192.168.1.86:1080 \
