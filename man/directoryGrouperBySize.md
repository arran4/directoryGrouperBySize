% directoryGrouperBySize(1)
# NAME

directoryGrouperBySize - group directory listings into disks of a target size

# SYNOPSIS

`directoryGrouperBySize -maxsize <size> [-f file]`

# DESCRIPTION

`directoryGrouperBySize` reads a list of directory sizes, typically from `du -sh`, and groups entries into virtual disks up to the specified size in gigabytes.

# OPTIONS

`-maxsize`  Maximum size in gigabytes for each disk (required).

`-f`  Path to file to read listing from. If omitted, standard input is used.

# EXAMPLE

`directoryGrouperBySize -maxsize 55 -f dirs.txt`

# SEE ALSO

`du(1)`
