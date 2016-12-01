package tagsearch

import (
	"crypto/sha256"
	"fmt"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/mapping"
	"github.com/blevesearch/blevex/rocksdb"
)

type Index struct {
	bleve.Index
}

type Entry struct {
	Language string `json:"language"`
	Kind     string `json:"kind"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Class    string `json:"class"`
	Inherits string `json:"inherits"`
	Access   string `json:"access"`
	Pattern  string `json:"pattern"`
}

func OpenRO(name string) (*Index, error) {
	val, err := bleve.OpenUsing(name, map[string]interface{}{
		"readonly": true,
	})
	if err != nil {
		return nil, err
	}
	return &Index{val}, nil
}

func OpenRW(name string) (*Index, error) {
	val, err := bleve.Open(name)
	if err == bleve.ErrorIndexPathDoesNotExist {
		val, err = bleve.NewUsing(name, indexMapping(), bleve.Config.DefaultIndexType, rocksdb.Name, nil)
		if err != nil {
			return nil, err
		}
	}

	return &Index{val}, nil
}

func (i *Index) TagCount() uint64 {
	count, _ := i.DocCount()
	return count
}

func (i *Index) SearchByName(name string, limit int) (*bleve.SearchResult, error) {
	query := bleve.NewMatchQuery(name)
	query.SetField("name")

	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Fields = []string{"*"}
	searchRequest.Size = limit
	// searchRequest.Explain = true

	return i.Search(searchRequest)
}

func (i *Index) Add(entry Entry) error {
	return i.Index.Index(entry.Key(), &entry)
}

func (i *Index) DeletePath(path string) error {
	query := bleve.NewMatchQuery(path)
	query.SetField("path")
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Size = 5000 // TODO: loop to delete >5k results

	searchResult, err := i.Search(searchRequest)
	if err != nil {
		return err
	}
	for _, hit := range searchResult.Hits {
		i.Delete(hit.ID)
	}
	return nil
}

func (e Entry) Key() string {
	name := e.FullName
	if name == "" {
		name = e.Name
	}
	key := fmt.Sprintf("%v-%v-%v-%v-%v", e.Language, e.Kind, e.Path, e.Line, name)
	sum := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", sum)
}

func indexMapping() mapping.IndexMapping {
	name := bleve.NewTextFieldMapping()
	name.Analyzer = keyword.Name
	name.IncludeInAll = true
	name.Store = true

	kw := bleve.NewTextFieldMapping()
	kw.Analyzer = keyword.Name
	kw.IncludeInAll = false
	kw.Store = true

	num := bleve.NewNumericFieldMapping()
	num.IncludeInAll = false
	num.Store = true

	stored := bleve.NewTextFieldMapping()
	stored.Index = false

	entry := bleve.NewDocumentStaticMapping()
	entry.AddFieldMappingsAt("kind", kw)
	entry.AddFieldMappingsAt("language", kw)
	entry.AddFieldMappingsAt("path", kw)
	entry.AddFieldMappingsAt("line", num)
	entry.AddFieldMappingsAt("pattern", stored)

	entry.AddFieldMappingsAt("name", name)
	entry.AddFieldMappingsAt("full_name", stored)
	entry.AddFieldMappingsAt("class", stored)
	entry.AddFieldMappingsAt("access", stored)
	entry.AddFieldMappingsAt("inherits", stored)

	mapping := bleve.NewIndexMapping()
	mapping.DefaultMapping = entry
	mapping.DefaultAnalyzer = "en"

	return mapping
}

