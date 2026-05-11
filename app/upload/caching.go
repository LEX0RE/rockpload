package upload

import (
	"bufio"
	"os"
	"strings"

	"github.com/LEX0RE/rockpload/app/constant"
	"github.com/LEX0RE/rockpload/app/tools/logger"
)

const (
	MAX_CACHE_SIZE_FACTOR = 2
	HISTORY_PER_ACCOUNT   = 20
)

type UploadCache struct {
	indexName string
	items     []string
	index     map[string]bool
	max       int
}

func LoadUploadedCache(indexName string, nAccount int) *UploadCache {
	logger.FuncDebug()
	c := &UploadCache{
		indexName: indexName,
		items:     []string{},
		index:     make(map[string]bool),
		max:       nAccount * HISTORY_PER_ACCOUNT * MAX_CACHE_SIZE_FACTOR,
	}

	file, err := os.Open(c.cachePath())
	if err != nil {
		if os.IsNotExist(err) {
			return c
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

		c.items = append(c.items, id)
		c.index[id] = true
	}

	return c
}

func (c *UploadCache) cachePath() string {
	return constant.UploadedCache + "_" + c.indexName
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

	for len(c.items) > c.max {
		old := c.items[0]
		c.items = c.items[1:]
		delete(c.index, old)
	}
}

func (c *UploadCache) Save() error {
	logger.FuncDebug()

	c.ensureCapacity()

	f, err := os.OpenFile(c.cachePath(), os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	for _, id := range c.items {
		f.WriteString(id + "\n")
	}

	return nil
}
