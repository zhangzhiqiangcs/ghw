//
// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.
//

package block

import (
	"fmt"
	"math"
	"strings"

	"github.com/jaypipes/ghw/pkg/context"
	"github.com/jaypipes/ghw/pkg/marshal"
	"github.com/jaypipes/ghw/pkg/option"
	"github.com/jaypipes/ghw/pkg/unitutil"
	"github.com/jaypipes/ghw/pkg/util"
)

// DriveType describes the general category of drive device
type DriveType int

const (
	DRIVE_TYPE_UNKNOWN DriveType = iota
	DRIVE_TYPE_HDD               // Hard disk drive
	DRIVE_TYPE_FDD               // Floppy disk drive
	DRIVE_TYPE_ODD               // Optical disk drive
	DRIVE_TYPE_SSD               // Solid-state drive
)

var (
	driveTypeString = map[DriveType]string{
		DRIVE_TYPE_UNKNOWN: "Unknown",
		DRIVE_TYPE_HDD:     "HDD",
		DRIVE_TYPE_FDD:     "FDD",
		DRIVE_TYPE_ODD:     "ODD",
		DRIVE_TYPE_SSD:     "SSD",
	}
)

func (dt DriveType) String() string {
	return driveTypeString[dt]
}

// NOTE(jaypipes): since serialized output is as "official" as we're going to
// get, let's lowercase the string output when serializing, in order to
// "normalize" the expected serialized output
func (dt DriveType) MarshalJSON() ([]byte, error) {
	return []byte("\"" + strings.ToLower(dt.String()) + "\""), nil
}

// StorageController is a category of block storage controller/driver. It
// represents more of the physical hardware interface than the storage
// protocol, which represents more of the software interface.
//
// See discussion on https://github.com/jaypipes/ghw/issues/117
type StorageController int

const (
	STORAGE_CONTROLLER_UNKNOWN StorageController = iota
	STORAGE_CONTROLLER_IDE                       // Integrated Drive Electronics
	STORAGE_CONTROLLER_SCSI                      // Small computer system interface
	STORAGE_CONTROLLER_NVME                      // Non-volatile Memory Express
	STORAGE_CONTROLLER_VIRTIO                    // Virtualized storage controller/driver
	STORAGE_CONTROLLER_MMC                       // Multi-media controller (used for mobile phone storage devices)
)

var (
	storageControllerString = map[StorageController]string{
		STORAGE_CONTROLLER_UNKNOWN: "Unknown",
		STORAGE_CONTROLLER_IDE:     "IDE",
		STORAGE_CONTROLLER_SCSI:    "SCSI",
		STORAGE_CONTROLLER_NVME:    "NVMe",
		STORAGE_CONTROLLER_VIRTIO:  "virtio",
		STORAGE_CONTROLLER_MMC:     "MMC",
	}
)

func (sc StorageController) String() string {
	return storageControllerString[sc]
}

// NOTE(jaypipes): since serialized output is as "official" as we're going to
// get, let's lowercase the string output when serializing, in order to
// "normalize" the expected serialized output
func (sc StorageController) MarshalJSON() ([]byte, error) {
	return []byte("\"" + strings.ToLower(sc.String()) + "\""), nil
}

// Disk describes a single disk drive on the host system. Disk drives provide
// raw block storage resources.
type Disk struct {
	Name                   string            `json:"name"`
	SizeBytes              uint64            `json:"size_bytes"`
	PhysicalBlockSizeBytes uint64            `json:"physical_block_size_bytes"`
	DriveType              DriveType         `json:"drive_type"`
	IsRemovable            bool              `json:"removable"`
	StorageController      StorageController `json:"storage_controller"`
	BusPath                string            `json:"bus_path"`
	// TODO(jaypipes): Convert this to a TopologyNode struct pointer and then
	// add to serialized output as "numa_node,omitempty"
	NUMANodeID   int          `json:"-"`
	Vendor       string       `json:"vendor"`
	Model        string       `json:"model"`
	SerialNumber string       `json:"serial_number"`
	WWN          string       `json:"wwn"`
	Partitions   []*Partition `json:"partitions"`
	MountInfo    *MountInfo   `json:"mount_info"`
	// TODO(jaypipes): Add PCI field for accessing PCI device information
	// PCI *PCIDevice `json:"pci"`
}

type MountInfo struct {
	MountPoint string `json:"mount_point"`
	Type       string `json:"type"`
	ReadOnly   bool   `json:"read_only"`
}

// Partition describes a logical division of a Disk.
type Partition struct {
	Disk      *Disk      `json:"-"`
	Name      string     `json:"name"`
	Label     string     `json:"label"`
	SizeBytes uint64     `json:"size_bytes"`
	UUID      string     `json:"uuid"` // This would be volume UUID on macOS, PartUUID on linux, empty on Windows
	MountInfo *MountInfo `json:"mount_info"`
}

// Info describes all disk drives and partitions in the host system.
type Info struct {
	ctx *context.Context
	// TODO(jaypipes): Deprecate this field and replace with TotalSizeBytes
	TotalPhysicalBytes uint64       `json:"total_size_bytes"`
	Disks              []*Disk      `json:"disks"`
	Partitions         []*Partition `json:"-"`
}

// New returns a pointer to an Info struct that describes the block storage
// resources of the host system.
func New(opts ...*option.Option) (*Info, error) {
	ctx := context.New(opts...)
	info := &Info{ctx: ctx}
	if err := ctx.Do(info.load); err != nil {
		return nil, err
	}
	return info, nil
}

func (i *Info) String() string {
	tpbs := util.UNKNOWN
	if i.TotalPhysicalBytes > 0 {
		tpb := i.TotalPhysicalBytes
		unit, unitStr := unitutil.AmountString(int64(tpb))
		tpb = uint64(math.Ceil(float64(tpb) / float64(unit)))
		tpbs = fmt.Sprintf("%d%s", tpb, unitStr)
	}
	dplural := "disks"
	if len(i.Disks) == 1 {
		dplural = "disk"
	}
	return fmt.Sprintf("block storage (%d %s, %s physical storage)",
		len(i.Disks), dplural, tpbs)
}

func (d *Disk) String() string {
	typeStr := ""
	mountStr := ""
	if d.MountInfo != nil {
		typeStr = fmt.Sprintf(" [%s]", d.MountInfo.Type)
		mountStr = fmt.Sprintf(" mounted@%s", d.MountInfo.MountPoint)
	}

	sizeStr := util.UNKNOWN
	if d.SizeBytes > 0 {
		size := d.SizeBytes
		unit, unitStr := unitutil.AmountString(int64(size))
		size = uint64(math.Ceil(float64(size) / float64(unit)))
		sizeStr = fmt.Sprintf("%d%s", size, unitStr)
	}
	atNode := ""
	if d.NUMANodeID >= 0 {
		atNode = fmt.Sprintf(" (node #%d)", d.NUMANodeID)
	}
	vendor := ""
	if d.Vendor != "" {
		vendor = " vendor=" + d.Vendor
	}
	model := ""
	if d.Model != util.UNKNOWN {
		model = " model=" + d.Model
	}
	serial := ""
	if d.SerialNumber != util.UNKNOWN {
		serial = " serial=" + d.SerialNumber
	}
	wwn := ""
	if d.WWN != util.UNKNOWN {
		wwn = " WWN=" + d.WWN
	}
	removable := ""
	if d.IsRemovable {
		removable = " removable=true"
	}
	return fmt.Sprintf(
		"%s %s (%s) %s%s%s [@%s%s]%s%s%s%s%s",
		d.Name,
		d.DriveType.String(),
		sizeStr,
		d.StorageController.String(),
		typeStr,
		mountStr,
		d.BusPath,
		atNode,
		vendor,
		model,
		serial,
		wwn,
		removable,
	)
}

func (p *Partition) String() string {
	typeStr := ""
	mountStr := ""
	if p.MountInfo != nil {
		typeStr = fmt.Sprintf("[%s]", p.MountInfo.Type)
		mountStr = fmt.Sprintf(" mounted@%s", p.MountInfo.MountPoint)
	}

	sizeStr := util.UNKNOWN
	if p.SizeBytes > 0 {
		size := p.SizeBytes
		unit, unitStr := unitutil.AmountString(int64(size))
		size = uint64(math.Ceil(float64(size) / float64(unit)))
		sizeStr = fmt.Sprintf("%d%s", size, unitStr)
	}
	return fmt.Sprintf(
		"%s (%s) %s%s",
		p.Name,
		sizeStr,
		typeStr,
		mountStr,
	)
}

// simple private struct used to encapsulate block information in a top-level
// "block" YAML/JSON map/object key
type blockPrinter struct {
	Info *Info `json:"block" yaml:"block"`
}

// YAMLString returns a string with the block information formatted as YAML
// under a top-level "block:" key
func (i *Info) YAMLString() string {
	return marshal.SafeYAML(i.ctx, blockPrinter{i})
}

// JSONString returns a string with the block information formatted as JSON
// under a top-level "block:" key
func (i *Info) JSONString(indent bool) string {
	return marshal.SafeJSON(i.ctx, blockPrinter{i}, indent)
}
