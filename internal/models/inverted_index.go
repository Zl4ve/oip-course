package models

type InvertedIndex struct {
	index map[string][]int
}

func NewInvertedIndex(index map[string][]int) *InvertedIndex {
	return &InvertedIndex{
		index: index,
	}
}

// Add добавляет для леммы номер страницы, в котором эта лемма встречается
func (ii *InvertedIndex) Add(lemma string, page int) {
	ii.index[lemma] = append(ii.index[lemma], page)
}

func (ii *InvertedIndex) GetIndex() map[string][]int {
	return ii.index
}
