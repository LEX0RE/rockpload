package upload

import (
	"bufio"
	"lexore/rockpload/app/config"
	"lexore/rockpload/app/tools/logger"
	"log/slog"
	"os"
	"strings"
)

const (
	maxCacheSize = 100
)

type UploadCache struct {
	items []string
	index map[string]bool
	max   int
}

func LoadUploadedCache() *UploadCache {
	logger.FuncDebug()
	cache := &UploadCache{
		items: []string{},
		index: make(map[string]bool),
		max: maxCacheSize,
	}

	file, err := os.Open(config.UploadedCache)
	if err != nil {
		if os.IsNotExist(err) {
			return cache
		}
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		id := strings.TrimSpace(scanner.Text())
		if id == "" {
			continue
		}

		cache.items = append(cache.items, id)
		cache.index[id] = true
	}

	logger.Rlogger.Debug("Storing cache in ", slog.Any("cachePath", config.GetCachePath()))

	return cache
}

func (c *UploadCache) Add(id string) {
	logger.FuncDebug()
	if c.index[id] {
		return
	}

	c.items = append(c.items, id)
	c.index[id] = true

	c.ensureCapacity()
	c.Save()
}

func (c *UploadCache) ensureCapacity() {
	logger.FuncDebug()

	for (len(c.items) > c.max) {
		old := c.items[0]
		c.items = c.items[1:]
		delete(c.index, old)
	}
}


func (c *UploadCache) Save() error {
	logger.FuncDebug()

	c.ensureCapacity()

	f, err := os.OpenFile(config.UploadedCache, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	for _, id := range c.items {
		f.WriteString(id + "\n")
	}

	logger.Rlogger.Debug("Cache file saved")

	return nil
}