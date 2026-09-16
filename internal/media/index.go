// Package media — index thread-safe en mémoire.
package media

import (
	"sync"
)

// Index stocke tous les médias en mémoire pour un accès ultra-rapide.
// Il est thread-safe et peut être lu/écrit depuis plusieurs goroutines simultanément.
type Index struct {
	mu     sync.RWMutex      // protège l'accès concurrent aux maps
	items  map[string]*Media // clé = ID, valeur = Media (accès rapide par ID)
	byPath map[string]*Media // clé = chemin absolu, valeur = Media (utile pour fsnotify)
}

// NewIndex crée un nouvel index vide.
func NewIndex() *Index {
	return &Index{
		items:  make(map[string]*Media),
		byPath: make(map[string]*Media),
	}
}

// Add ajoute ou met à jour un média dans l'index.
// Si un média avec le même ID existe déjà, il est remplacé.
func (idx *Index) Add(m *Media) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.items[m.ID] = m
	idx.byPath[m.Path] = m
}

// Remove supprime un média de l'index par son chemin.
// Retourne true si le média existait et a été supprimé.
func (idx *Index) Remove(path string) bool {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	media, exists := idx.byPath[path]
	if !exists {
		return false
	}

	delete(idx.items, media.ID)
	delete(idx.byPath, path)
	return true
}

// Get récupère un média par son ID.
// Retourne nil si le média n'existe pas.
func (idx *Index) Get(id string) *Media {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.items[id]
}

// GetByPath récupère un média par son chemin absolu.
// Retourne nil si le média n'existe pas.
func (idx *Index) GetByPath(path string) *Media {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.byPath[path]
}

// All retourne une copie de tous les médias indexés.
// La copie est nécessaire pour éviter les race conditions si l'appelant
// parcourt la slice pendant qu'un autre thread modifie l'index.
func (idx *Index) All() []*Media {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	result := make([]*Media, 0, len(idx.items))
	for _, m := range idx.items {
		result = append(result, m)
	}
	return result
}

// Count retourne le nombre total de médias indexés.
func (idx *Index) Count() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return len(idx.items)
}

// Clear vide complètement l'index.
func (idx *Index) Clear() {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.items = make(map[string]*Media)
	idx.byPath = make(map[string]*Media)
}

// GetByFolder retourne tous les médias d'un dossier spécifique, triés par nom.
// Le tri alphabétique garantit l'ordre des épisodes d'une série.
func (idx *Index) GetByFolder(folder string) []*Media {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var result []*Media
	for _, m := range idx.items {
		if m.Folder == folder {
			result = append(result, m)
		}
	}

	// Tri alphabétique par nom de fichier
	// Utilise une simple bubble sort (suffisant pour des petites listes)
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].Name > result[j].Name {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// GetNext retourne le média suivant dans le même dossier (ordre alphabétique).
// Retourne nil si le média est le dernier du dossier ou si le média n'existe pas.
func (idx *Index) GetNext(id string) *Media {
	current := idx.Get(id)
	if current == nil {
		return nil
	}

	// Récupère tous les médias du même dossier
	siblings := idx.GetByFolder(current.Folder)
	if len(siblings) <= 1 {
		return nil // un seul fichier dans le dossier
	}

	// Trouve la position actuelle dans la liste
	for i, m := range siblings {
		if m.ID == id {
			// Si c'est le dernier, retourne nil
			if i == len(siblings)-1 {
				return nil
			}
			// Sinon retourne le suivant
			return siblings[i+1]
		}
	}

	return nil
}
