//
// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.
//

// +build linux

package block

import (
	"os"
	"reflect"
	"testing"
)

func TestParseMountEntry(t *testing.T) {
	if _, ok := os.LookupEnv("GHW_TESTING_SKIP_BLOCK"); ok {
		t.Skip("Skipping block tests.")
	}

	tests := []struct {
		line     string
		expected *mountEntry
	}{
		{
			line: "/dev/sda6 / ext4 rw,relatime,errors=remount-ro,data=ordered 0 0",
			expected: &mountEntry{
				Device:         "/dev/sda6",
				Mountpoint:     "/",
				FilesystemType: "ext4",
				Options: []string{
					"rw",
					"relatime",
					"errors=remount-ro",
					"data=ordered",
				},
			},
		},
		{
			line: "/dev/sda8 /home/Name\\040with\\040spaces ext4 ro 0 0",
			expected: &mountEntry{
				Device:         "/dev/sda8",
				Mountpoint:     "/home/Name with spaces",
				FilesystemType: "ext4",
				Options: []string{
					"ro",
				},
			},
		},
		{
			// Whoever might do this in real life should be quarantined and
			// placed in administrative segregation
			line: "/dev/sda8 /home/Name\\011with\\012tab&newline ext4 ro 0 0",
			expected: &mountEntry{
				Device:         "/dev/sda8",
				Mountpoint:     "/home/Name\twith\ntab&newline",
				FilesystemType: "ext4",
				Options: []string{
					"ro",
				},
			},
		},
		{
			line: "/dev/sda1 /home/Name\\\\withslash ext4 ro 0 0",
			expected: &mountEntry{
				Device:         "/dev/sda1",
				Mountpoint:     "/home/Name\\withslash",
				FilesystemType: "ext4",
				Options: []string{
					"ro",
				},
			},
		},
		{
			line:     "Indy, bad dates",
			expected: nil,
		},
	}

	for x, test := range tests {
		actual := parseMountEntry(test.line)
		if test.expected == nil {
			if actual != nil {
				t.Fatalf("Expected nil, but got %v", actual)
			}
		} else if !reflect.DeepEqual(test.expected, actual) {
			t.Fatalf("In test %d, expected %v == %v", x, test.expected, actual)
		}
	}
}

func TestDiskTypes(t *testing.T) {
	if _, ok := os.LookupEnv("GHW_TESTING_SKIP_BLOCK"); ok {
		t.Skip("Skipping block tests.")
	}

	type entry struct {
		driveType         DriveType
		storageController StorageController
	}

	tests := []struct {
		line     string
		expected entry
	}{
		{
			line: "sda6",
			expected: entry{
				driveType:         DRIVE_TYPE_HDD,
				storageController: STORAGE_CONTROLLER_SCSI,
			},
		},
		{
			line: "nvme0n1",
			expected: entry{
				driveType:         DRIVE_TYPE_SSD,
				storageController: STORAGE_CONTROLLER_NVME,
			},
		},
		{
			line: "vda1",
			expected: entry{
				driveType:         DRIVE_TYPE_HDD,
				storageController: STORAGE_CONTROLLER_VIRTIO,
			},
		},
		{
			line: "xvda1",
			expected: entry{
				driveType:         DRIVE_TYPE_HDD,
				storageController: STORAGE_CONTROLLER_SCSI,
			},
		},
		{
			line: "fda1",
			expected: entry{
				driveType:         DRIVE_TYPE_FDD,
				storageController: STORAGE_CONTROLLER_UNKNOWN,
			},
		},
		{
			line: "sr0",
			expected: entry{
				driveType:         DRIVE_TYPE_ODD,
				storageController: STORAGE_CONTROLLER_SCSI,
			},
		},
		{
			line: "mmcblk0",
			expected: entry{
				driveType:         DRIVE_TYPE_SSD,
				storageController: STORAGE_CONTROLLER_MMC,
			},
		},
		{
			line: "Indy, bad dates",
			expected: entry{
				driveType:         DRIVE_TYPE_UNKNOWN,
				storageController: STORAGE_CONTROLLER_UNKNOWN,
			},
		},
	}

	for _, test := range tests {
		gotDriveType, gotStorageController := diskTypes(test.line)
		if test.expected.driveType != gotDriveType {
			t.Fatalf(
				"For %s, expected drive type %s, but got %s",
				test.line, test.expected.driveType, gotDriveType,
			)
		}
		if test.expected.storageController != gotStorageController {
			t.Fatalf(
				"For %s, expected storage controller %s, but got %s",
				test.line, test.expected.storageController, gotStorageController,
			)
		}
	}
}
