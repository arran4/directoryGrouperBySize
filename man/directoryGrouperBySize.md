% directoryGrouperBySize(1)
# NAME

directoryGrouperBySize - group directory listings into disks of a target size

# SYNOPSIS

`directoryGrouperBySize -maxsize <size> [-f file] [-scan dir]`

# DESCRIPTION

`directoryGrouperBySize` reads a list of directory sizes, typically from `du -sh`, and groups entries into virtual disks up to the specified size in gigabytes.

# OPTIONS

`-maxsize`  Maximum size for each disk. Accepts units (G, M, K, T) and defaults to gigabytes when omitted. (required).

`-f`  Path to file to read listing from. If omitted, standard input is used.

`-scan`  Run `du -sh` on the specified directory instead of reading input.

# EXAMPLE

`directoryGrouperBySize -maxsize 55G -f dirs.txt`

`directoryGrouperBySize -maxsize 55G -scan /media`

# SEE ALSO

`du(1)`
