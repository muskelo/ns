package filemanager

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

type File struct {
	Name string
	Path string
}

type Directory struct {
	Name string
	Path string
}

type FileManager struct {
	Root string
}

func (fm *FileManager) ReadDir(path string) ([]File, []Directory, error) {
	entriesList, err := os.ReadDir(fm.Full(path))
	if err != nil {
		return nil, nil, err
	}

	filesList := make([]File, 0)
	dirsList := make([]Directory, 0)
	for _, entry := range entriesList {
		if entry.IsDir() {
			dirsList = append(dirsList, Directory{
				Name: entry.Name(),
				Path: filepath.Join(path, entry.Name()),
			})
		} else {
			filesList = append(filesList, File{
				Name: entry.Name(),
				Path: filepath.Join(path, entry.Name()),
			})
		}
	}
	return filesList, dirsList, nil
}

func (fm *FileManager) Mkdir(path string) error {
	return os.Mkdir(fm.Full(path), 0770)
}

func (fm *FileManager) Open(path string) (*os.File, error) {
	return os.Open(fm.Full(path))
}

func (fm *FileManager) Create(path string) (*os.File, error) {
	return os.Create(fm.Full(path))
}

func (fm *FileManager) Remove(path string) error {
	return os.Remove(fm.Full(path))
}

// return full path
func (fm *FileManager) Full(path string) string {
	return filepath.Join(fm.Root, path)
}


func (fm *FileManager) Stat(path string) (fs.FileInfo, bool, error) {
    info, err := os.Stat(fm.Full(path))
    if err == nil {
        return info, true, nil
    }
    if errors.Is(err, os.ErrNotExist){
        return nil, false, nil
    } else {
        return nil, false, err
    }
}
func (fm *FileManager) IsDirExist(path string) (bool, error) {
	stat, err := os.Stat(fm.Full(path))
	switch {
	case err == nil:
		return stat.IsDir(), nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, err
	}
}
func (fm *FileManager) IsFileExist(path string) (bool, error) {
	stat, err := os.Stat(fm.Full(path))
	switch {
	case err == nil:
		return !stat.IsDir(), nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, err
	}
}
func (fm *FileManager) IsExist(path string) (bool, error) {
    _, err := os.Stat(fm.Full(path))
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, err
	}
}
