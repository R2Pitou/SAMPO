//go:build windows

package rootidentity

import (
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"

	"sampo/internal/domain"
)

const (
	volumeNameGUID = 0x1
	fileNameOpened = 0x8
)

type fileIDInfo struct {
	VolumeSerialNumber uint64
	FileID             [16]byte
}

func (SystemProber) Probe(submittedLocator string) (domain.ProviderRoot, error) {
	if !filepath.IsAbs(submittedLocator) {
		return domain.ProviderRoot{}, errors.New("provider root must be an absolute path")
	}
	path, err := windows.UTF16PtrFromString(submittedLocator)
	if err != nil {
		return domain.ProviderRoot{}, fmt.Errorf("encode provider root: %w", err)
	}
	handle, err := windows.CreateFile(path, windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return domain.ProviderRoot{}, fmt.Errorf("open provider root: %w", err)
	}
	defer windows.CloseHandle(handle)

	var fallback windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &fallback); err != nil {
		return domain.ProviderRoot{}, fmt.Errorf("inspect provider root: %w", err)
	}
	if fallback.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0 {
		return domain.ProviderRoot{}, errors.New("provider root must be a directory")
	}

	operational, err := finalPath(handle, 0)
	if err != nil {
		operational, err = finalPath(handle, fileNameOpened)
		if err != nil {
			return domain.ProviderRoot{}, fmt.Errorf("resolve operational provider root: %w", err)
		}
	}
	finalEvidence, guidErr := finalPath(handle, volumeNameGUID)
	if guidErr != nil {
		finalEvidence = operational
	}

	fallbackIdentity := fmt.Sprintf("windows:file-index64:%08x:%08x%08x",
		fallback.VolumeSerialNumber, fallback.FileIndexHigh, fallback.FileIndexLow)
	root := domain.ProviderRoot{
		SubmittedLocator:   submittedLocator,
		OperationalLocator: operational,
		FinalPathEvidence:  finalEvidence,
		FallbackIdentity:   fallbackIdentity,
		IdentityConfidence: domain.RootIdentityFallback,
	}

	var identity fileIDInfo
	if err := windows.GetFileInformationByHandleEx(handle, windows.FileIdInfo,
		(*byte)(unsafe.Pointer(&identity)), uint32(unsafe.Sizeof(identity))); err == nil {
		root.PhysicalIdentity = fmt.Sprintf("windows:file-id128:%016x:%s",
			identity.VolumeSerialNumber, hex.EncodeToString(identity.FileID[:]))
		root.IdentityConfidence = domain.RootIdentityStrong
	}
	if isRemotePath(operational) {
		root.IdentityConfidence = domain.RootIdentityWeak
		root.CatalogueOnly = true
	}
	return root, nil
}

func finalPath(handle windows.Handle, flags uint32) (string, error) {
	size := uint32(256)
	for {
		buffer := make([]uint16, size)
		n, err := windows.GetFinalPathNameByHandle(handle, &buffer[0], size, flags)
		if err != nil {
			return "", err
		}
		if n < size {
			return windows.UTF16ToString(buffer[:n]), nil
		}
		size = n + 1
	}
}

func isRemotePath(path string) bool {
	if strings.HasPrefix(strings.ToUpper(path), `\\?\UNC\`) {
		return true
	}
	if strings.HasPrefix(path, `\\`) && !strings.HasPrefix(path, `\\?\`) {
		return true
	}
	volume := filepath.VolumeName(path)
	if volume == "" {
		return false
	}
	root := volume + `\`
	ptr, err := windows.UTF16PtrFromString(root)
	return err == nil && windows.GetDriveType(ptr) == windows.DRIVE_REMOTE
}
