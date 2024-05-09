package media

import (
	"context"
	"github.com/fsnotify/fsnotify"
	log "go.uber.org/zap"
)

func (m *Media) ProcessFileAsMedia(ctx context.Context, path string) error {
	m.Lock()
	defer m.Unlock()
	// Get the fileinfo
	//fileInfo, err := os.Stat(path)
	//if err != nil {
	//	return err
	//}
	// Gives the modification time
	// modificationTime := fileInfo.ModTime()

	metaData, err := getMetaData(ctx, path)
	if err == nil {
		m.allFiles = append(m.allFiles, File{Path: path, MetaData: *metaData})
		log.S().Debugf("%s, %+v", path, metaData)
	}
	return err
}

func (m *Media) QueueFile(path string) {
	m.pendingFilePaths <- path
}

func (m *Media) QueueFileChange(event fsnotify.Event) {
	m.watchEvents <- event
}

func (m *Media) RemoveFile(path string) {
	m.Lock()
	defer m.Unlock()
	for i, v := range m.allFiles {
		if v.Path == path {
			m.allFiles = append(m.allFiles[:i], m.allFiles[i+1:]...)
			break
		}
	}
	for i, v := range m.newFiles {
		if v.Path == path {
			m.newFiles = append(m.newFiles[:i], m.newFiles[i+1:]...)
			break
		}
	}
	for i, v := range m.unseenFiles {
		if v.Path == path {
			m.unseenFiles = append(m.unseenFiles[:i], m.unseenFiles[i+1:]...)
			break
		}
	}
}

func (m *Media) Worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-m.watchEvents:
			if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				m.RemoveFile(event.Name)
				continue
			}

			if !event.Has(fsnotify.Create) && !event.Has(fsnotify.Write) {
				continue
			}
			// TODO: Insert new files at the beginning of a queue
			if err := m.ProcessFileAsMedia(ctx, event.Name); err != nil {
				log.S().Error(err)
			}
		case path := <-m.pendingFilePaths:
			if err := m.ProcessFileAsMedia(ctx, path); err != nil {
				log.S().Error(err)
			}
		}
	}
}
