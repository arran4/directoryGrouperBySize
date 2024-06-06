Takes a `du -sh` output, and then groups it into "disks" of a particular size.

# Download / Install

See: https://github.com/arran4/directoryGrouperBySize/releases For downloadable and installable versions.

# Example

Such as:

```
3.6G    FileFolder1
1.6G    FileFolder2
27G     FileFolder3
52G     FileFolder4
894M    FileFolder5
7.5G    FileFolder6
11G     FileFolder7
2.3G    FileFolder8
8.7G    FileFolder9
13G     FileFolder10
3.3G    FileFolder11
1.7G    FileFolder12
5.1G    FileFolder13
4.3G    FileFolder14
```

Becomes:
```
## Disk 1 (32.20 GB used, 22.80 GB free)
FileFolder1
FileFolder2
FileFolder3

## Disk 2 (52.87 GB used, 2.13 GB free)
FileFolder4
FileFolder5

## Disk 3 (52.60 GB used, 2.40 GB free)
FileFolder6
FileFolder7
FileFolder8
FileFolder9
FileFolder10
FileFolder11
FileFolder12
FileFolder13

## Disk 4 (4.30 GB used, 50.70 GB free)
FileFolder14
```

It's done by order of input. It shouldn't be too hard to add a best fit.
