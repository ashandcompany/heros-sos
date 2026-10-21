package avatar

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"

	"github.com/google/uuid"
)

const (
	fileExtension = ".jpg"
	// directoryMode et fileMode : lisibles par le propriétaire et son groupe seulement.
	directoryMode os.FileMode = 0o750
	fileMode      os.FileMode = 0o640
)

// namePattern est la forme d'un nom de fichier produit par Save : un UUID aléatoire. Toute autre
// forme (séparateur de chemin, « .. », extension) est refusée avant tout accès au disque.
var namePattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}` + regexp.QuoteMeta(fileExtension) + `$`)

// ErrNotFound indique qu'aucune photo ne porte ce nom.
var ErrNotFound = errors.New("avatar : photo introuvable")

// IsValidName indique si name a la forme d'un nom de fichier produit par Save.
func IsValidName(name string) bool { return namePattern.MatchString(name) }

// DiskStore écrit les photos dans un dossier. Il n'accède qu'à ce dossier (os.Root) : même un nom
// forgé ne pourrait pas en sortir. Le nom d'une photo est un UUID aléatoire, jamais l'identifiant
// du compte : l'adresse d'une photo ne se devine pas, et en changer à chaque envoi invalide les
// caches du navigateur.
//
// Le dossier est ouvert à chaque opération plutôt que gardé ouvert : il n'y a ainsi aucune ressource
// à fermer à l'arrêt, pour le coût d'un appel système.
type DiskStore struct {
	dir string
}

// NewDiskStore prépare le dossier des photos, en le créant s'il n'existe pas.
func NewDiskStore(dir string) (*DiskStore, error) {
	if err := os.MkdirAll(dir, directoryMode); err != nil {
		return nil, fmt.Errorf("avatar : création du dossier %q : %w", dir, err)
	}
	store := &DiskStore{dir: dir}
	root, err := store.openRoot()
	if err != nil {
		return nil, err
	}
	return store, root.Close()
}

func (s *DiskStore) openRoot() (*os.Root, error) {
	root, err := os.OpenRoot(s.dir)
	if err != nil {
		return nil, fmt.Errorf("avatar : ouverture du dossier %q : %w", s.dir, err)
	}
	return root, nil
}

// Save écrit une photo sous un nouveau nom et retourne ce nom. Un fichier à moitié écrit est supprimé.
func (s *DiskStore) Save(data []byte) (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("avatar : tirage du nom : %w", err)
	}
	name := id.String() + fileExtension

	root, err := s.openRoot()
	if err != nil {
		return "", err
	}
	defer func() { _ = root.Close() }()

	// O_EXCL : ne jamais écraser une photo existante.
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fileMode)
	if err != nil {
		return "", fmt.Errorf("avatar : création du fichier : %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = root.Remove(name)
		return "", fmt.Errorf("avatar : écriture du fichier : %w", err)
	}
	if err := file.Close(); err != nil {
		_ = root.Remove(name)
		return "", fmt.Errorf("avatar : fermeture du fichier : %w", err)
	}
	return name, nil
}

// Open ouvre une photo. L'appelant ferme le fichier.
func (s *DiskStore) Open(name string) (*os.File, error) {
	if !IsValidName(name) {
		return nil, ErrNotFound
	}
	root, err := s.openRoot()
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }() // le fichier ouvert reste utilisable après la fermeture du dossier

	file, err := root.Open(name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("avatar : ouverture de %s : %w", name, err)
	}
	return file, nil
}

// Delete supprime une photo. Une photo déjà absente n'est pas une erreur : la suppression est
// idempotente.
func (s *DiskStore) Delete(name string) error {
	if !IsValidName(name) {
		return ErrNotFound
	}
	root, err := s.openRoot()
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	if err := root.Remove(name); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("avatar : suppression de %s : %w", name, err)
	}
	return nil
}
