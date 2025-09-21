# Erasure Coding with Reed-Solomon

This repository is for learning erasure coding concepts. You can test how Reed-Solomon algorithm recovers data even when data blocks are lost through these simple steps.

## Requirements

- Go 1.24 or later
- Linux with `sudo` access
- `xfsprogs` package (for XFS filesystem)

## Quick Start

Follow these steps to learn how Reed-Solomon algorithm recovers data from lost blocks:

```bash
# 1. Setup - Create 6 filesystems under .workdir
./scripts/disk.sh prepare .workdir 6

# 2. Save file - Split into 4 data blocks and 2 parity blocks
go run main.go save ./examples/sample.txt

# 3. Simulate disk failure
umount .workdir/mnt/disk1

# 4. Load file even with one disk missing
go run main.go load ./examples/sample.txt

# 5. Cleanup all test disks
./scripts/disk.sh cleanup .workdir
```

## Missing Features

This repository is for learning purposes, so many features are missing:

- Chunk processing: Loads the entire target file into memory at once
- BitRot protection: No checksum verification during write or read operations
- Flexible disk selection: Only supports fixed format disk names
- Disk health monitoring: No disk failure detection or health checks

## Dependencies

- [klauspost/reedsolomon](https://github.com/klauspost/reedsolomon) - Reed-Solomon implementation
- [spf13/cobra](https://github.com/spf13/cobra) - CLI framework