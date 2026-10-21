// Package avatar traite les photos de profil : contrôle et normalisation de l'image reçue, stockage
// sur disque, et service du fichier.
//
// Une photo n'est jamais conservée telle que reçue. Elle est décodée puis ré-encodée en JPEG : cela
// prouve que c'est bien une image, efface les métadonnées (position GPS, appareil, date) et tout
// contenu ajouté après l'image, et donne un format unique, donc un type de contenu fixe à servir.
package avatar

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png" // enregistre le décodeur PNG pour image.Decode
	"mime"
	"strings"

	_ "golang.org/x/image/webp" // enregistre le décodeur WebP : le frontend envoie du WebP par défaut
)

// Types de contenu acceptés à l'envoi.
const (
	ContentTypeJPEG = "image/jpeg"
	ContentTypePNG  = "image/png"
	ContentTypeWebP = "image/webp"

	// StoredContentType est le format de toutes les photos stockées.
	StoredContentType = ContentTypeJPEG

	// jpegQuality est la qualité du ré-encodage : une photo de profil de quelques centaines de pixels
	// reste nette et pèse quelques dizaines de Kio.
	jpegQuality = 85
)

var (
	// ErrInvalidImage indique un fichier qui n'est pas une image JPEG, PNG ou WebP lisible.
	ErrInvalidImage = errors.New("avatar : image invalide")
	// ErrTooLarge indique une image dont une dimension dépasse la limite.
	ErrTooLarge = errors.New("avatar : image trop grande")
)

// IsAcceptedContentType indique si le type de contenu déclaré à l'envoi est accepté. Les paramètres
// éventuels (« ; charset=… ») sont ignorés. Le type déclaré n'est qu'un filtre : seul le contenu
// décodé fait foi.
func IsAcceptedContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	switch strings.ToLower(mediaType) {
	case ContentTypeJPEG, ContentTypePNG, ContentTypeWebP:
		return true
	}
	return false
}

// Normalize vérifie qu'un fichier est une image JPEG, PNG ou WebP dont les dimensions ne dépassent
// pas maxDimension, puis la ré-encode en JPEG sur fond blanc. Les dimensions sont lues avant le
// décodage complet : une image minuscule en octets mais gigantesque en pixels est refusée sans
// allouer sa mémoire.
func Normalize(data []byte, maxDimension int) ([]byte, error) {
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w : %v", ErrInvalidImage, err)
	}
	switch format {
	case "jpeg", "png", "webp":
	default:
		return nil, fmt.Errorf("%w : format %q", ErrInvalidImage, format)
	}
	if config.Width <= 0 || config.Height <= 0 {
		return nil, fmt.Errorf("%w : dimensions nulles", ErrInvalidImage)
	}
	if config.Width > maxDimension || config.Height > maxDimension {
		return nil, ErrTooLarge
	}

	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w : %v", ErrInvalidImage, err)
	}

	// Le JPEG n'a pas de transparence : une image PNG ou WebP transparente est posée sur du blanc
	// plutôt que de devenir noire.
	bounds := decoded.Bounds()
	flattened := image.NewRGBA(bounds)
	draw.Draw(flattened, bounds, image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(flattened, bounds, decoded, bounds.Min, draw.Over)

	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, flattened, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, fmt.Errorf("avatar : encodage JPEG : %w", err)
	}
	return encoded.Bytes(), nil
}
