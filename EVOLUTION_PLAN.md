# Evolution Plan: From Media Converter to Media Manager

**Date:** 2025-11-01
**Status:** Planning Phase
**Target:** Version 2.0

---

## 1. Nouvelle Vision et Identité du Projet

### 1.1 Vision Statement

> **"Organisateur et archiveur de médias avec conversion optionnelle"**

Le projet évolue d'un simple convertisseur de médias vers un gestionnaire complet de workflow photographique et vidéo, couvrant deux besoins complémentaires :
1. **Archivage sécurisé** : Sauvegarde intelligente sur NAS avec organisation par date
2. **Optimisation** : Conversion vers formats modernes pour web/partage

### 1.2 Propositions de Noms

#### Option 1: **MediaFlow** ⭐ (Recommandé)
- **Signification:** Gestion du flux complet des médias (capture → archive → optimisation)
- **Avantages:** Court, international, évoque le mouvement et le workflow
- **Commande:** `mediaflow --copy-only /source /nas`

#### Option 2: **MediaVault**
- **Signification:** Coffre-fort pour médias avec organisation automatique
- **Avantages:** Évoque sécurité et archivage
- **Inconvénient:** Moins évocateur de la fonction conversion

#### Option 3: **ArchiveFlow**
- **Signification:** Flux d'archivage intelligent
- **Avantages:** Clair sur la fonction principale
- **Inconvénient:** Masque l'aspect conversion

#### Option 4: **MediaOrganizer**
- **Signification:** Descriptif direct
- **Avantages:** Très clair, SEO-friendly
- **Inconvénient:** Un peu générique

#### Option 5: **FluxMedia**
- **Signification:** Variante française de MediaFlow
- **Avantages:** Sonorité élégante
- **Inconvénient:** Moins international

**Recommandation finale:** **MediaFlow** - combine l'essence du workflow avec une connotation moderne et professionnelle.

---

## 2. Architecture Technique

### 2.1 Nouveau Mode: `--copy-only`

#### 2.1.1 Comportement

```bash
# Mode archivage NAS (copie pure)
mediaflow --copy-only /Volumes/SD-Card /Volumes/NAS/Photos-Master

# Mode conversion (actuel, comportement par défaut)
mediaflow /Volumes/NAS/Photos-Master/2024 ~/Photos-Optimized

# Erreur si flags incompatibles
mediaflow --copy-only --quality 85 /source /dest
# ❌ Error: --quality flag cannot be used with --copy-only mode
```

#### 2.1.2 Flags Incompatibles avec --copy-only

Les flags suivants retournent une erreur explicite en mode `--copy-only` :
- `--quality`
- `--format` (avif/webp)
- `--video-codec` (h265/h264/av1)
- `--video-quality`

Flags compatibles :
- `--organize-by-date` (organisation par date)
- `--language` (noms de mois)
- `--keep-originals` (n/a en copy-only, mais ignoré silencieusement)
- `--workers` (parallélisation des copies)
- `--dry-run` (simulation)

#### 2.1.3 Flux de Traitement

```
┌─────────────────────────────────────────────────────┐
│                  Source Directory                    │
│            (Carte SD / Disque externe)               │
└───────────────────┬─────────────────────────────────┘
                    │
                    ▼
          ┌─────────────────────┐
          │  1. Scan & Filter   │
          │  - Skip system files│
          │  - Detect file types│
          └──────────┬──────────┘
                     │
                     ▼
          ┌─────────────────────┐
          │ 2. Extract Metadata │
          │  - Get file date    │
          │  - Generate dest    │
          └──────────┬──────────┘
                     │
                     ▼
          ┌─────────────────────┐
          │ 3. Check Duplicate  │
          │  - Calc xxHash64    │
          │  - Compare checksum │
          └──────────┬──────────┘
                     │
        ┌────────────┴────────────┐
        │                         │
        ▼                         ▼
  ┌──────────┐            ┌──────────────┐
  │  EXISTS  │            │  NOT EXISTS  │
  │  + SAME  │            │  OR DIFFER   │
  │ CHECKSUM │            │              │
  └────┬─────┘            └──────┬───────┘
       │                         │
       ▼                         ▼
  ┌─────────┐            ┌──────────────┐
  │  SKIP   │            │ 4. Copy File │
  │  + LOG  │            │   (io.Copy)  │
  └─────────┘            └──────┬───────┘
                                │
                                ▼
                      ┌──────────────────┐
                      │ 5. Verify Copy   │
                      │  - Calc checksum │
                      │  - Compare       │
                      └────────┬─────────┘
                               │
                  ┌────────────┴────────────┐
                  │                         │
                  ▼                         ▼
          ┌──────────────┐         ┌──────────────┐
          │   SUCCESS    │         │    FAILED    │
          │  Keep copy   │         │  Delete copy │
          │  Log success │         │  Log error   │
          └──────────────┘         └──────────────┘
```

### 2.2 Système de Checksum (xxHash64)

#### 2.2.1 Pourquoi xxHash64 ?

| Algorithme | Vitesse (Go/s) | Taille | Collision | Usage |
|-----------|----------------|--------|-----------|-------|
| **xxHash64** | ~10 GB/s | 64-bit | Extrêmement rare | **✅ Choisi** |
| MD5 | ~0.5 GB/s | 128-bit | Rare | Obsolète (sécurité) |
| SHA256 | ~0.2 GB/s | 256-bit | Très rare | Trop lent pour volumes |
| CRC32 | ~15 GB/s | 32-bit | Possible | Trop de collisions |

**Justification:** xxHash64 offre le meilleur compromis vitesse/fiabilité pour la détection de doublons et la vérification d'intégrité sur de gros volumes.

#### 2.2.2 Nouvelle Package Structure

```
internal/
├── checksum/
│   ├── checksum.go          # Interface et implémentation xxHash64
│   ├── checksum_test.go     # Tests unitaires
│   └── tracker.go           # Tracking des checksums en mémoire
```

#### 2.2.3 Interface Checksum

```go
// internal/checksum/checksum.go
package checksum

import (
    "fmt"
    "io"
    "os"
    "github.com/cespare/xxhash/v2"
)

// Hasher calculates file checksums
type Hasher interface {
    Calculate(filePath string) (uint64, error)
    CalculateReader(reader io.Reader) (uint64, error)
}

// XXHash64Hasher implements Hasher using xxHash64
type XXHash64Hasher struct{}

func NewXXHash64Hasher() *XXHash64Hasher {
    return &XXHash64Hasher{}
}

// Calculate computes xxHash64 for a file
func (h *XXHash64Hasher) Calculate(filePath string) (uint64, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return 0, fmt.Errorf("failed to open file: %w", err)
    }
    defer file.Close()

    return h.CalculateReader(file)
}

// CalculateReader computes xxHash64 from a reader
func (h *XXHash64Hasher) CalculateReader(reader io.Reader) (uint64, error) {
    hasher := xxhash.New()

    // Use 32KB buffer for efficient reading
    buf := make([]byte, 32*1024)

    if _, err := io.CopyBuffer(hasher, reader, buf); err != nil {
        return 0, fmt.Errorf("failed to read data: %w", err)
    }

    return hasher.Sum64(), nil
}

// FormatChecksum returns human-readable hex representation
func FormatChecksum(checksum uint64) string {
    return fmt.Sprintf("%016x", checksum)
}
```

#### 2.2.4 Checksum Tracker

```go
// internal/checksum/tracker.go
package checksum

import (
    "sync"
)

// Tracker keeps track of file checksums during a session
type Tracker struct {
    mu        sync.RWMutex
    checksums map[uint64][]string // checksum -> list of file paths
}

func NewTracker() *Tracker {
    return &Tracker{
        checksums: make(map[uint64][]string),
    }
}

// Register adds a file checksum to the tracker
func (t *Tracker) Register(checksum uint64, filePath string) {
    t.mu.Lock()
    defer t.mu.Unlock()

    t.checksums[checksum] = append(t.checksums[checksum], filePath)
}

// IsDuplicate checks if a checksum already exists
func (t *Tracker) IsDuplicate(checksum uint64) bool {
    t.mu.RLock()
    defer t.mu.RUnlock()

    paths, exists := t.checksums[checksum]
    return exists && len(paths) > 0
}

// GetOriginalPath returns the first file with this checksum
func (t *Tracker) GetOriginalPath(checksum uint64) (string, bool) {
    t.mu.RLock()
    defer t.mu.RUnlock()

    paths, exists := t.checksums[checksum]
    if !exists || len(paths) == 0 {
        return "", false
    }
    return paths[0], true
}

// Stats returns tracker statistics
func (t *Tracker) Stats() (uniqueFiles int, duplicates int) {
    t.mu.RLock()
    defer t.mu.RUnlock()

    uniqueFiles = len(t.checksums)
    totalFiles := 0
    for _, paths := range t.checksums {
        totalFiles += len(paths)
    }
    duplicates = totalFiles - uniqueFiles

    return
}
```

### 2.3 Détection de Doublons

#### 2.3.1 Logique de Détection

**Comportement:**
1. Avant chaque copie, calculer xxHash64 du fichier source
2. Vérifier si un fichier avec ce checksum existe déjà dans la destination
3. Si oui : skip + log
4. Si non : copier et ajouter au tracker

**Portée:** Détection uniquement entre source et destination (pas de scan interne à la source ou à la destination existante)

#### 2.3.2 Implémentation

```go
// internal/converter/copy_mode.go (nouveau fichier)
package converter

import (
    "fmt"
    "io"
    "os"
    "path/filepath"
    "github.com/yourusername/mediaflow/internal/checksum"
    "github.com/yourusername/mediaflow/internal/logger"
)

// CopyFile copies a file with checksum verification
func (c *Converter) CopyFile(srcPath, destPath string) error {
    // 1. Calculate source checksum
    srcChecksum, err := c.hasher.Calculate(srcPath)
    if err != nil {
        return fmt.Errorf("failed to calculate source checksum: %w", err)
    }

    // 2. Check if destination file exists
    if _, err := os.Stat(destPath); err == nil {
        // File exists, check if it's the same
        destChecksum, err := c.hasher.Calculate(destPath)
        if err != nil {
            logger.Log.Warnf("Failed to verify existing file %s: %v", destPath, err)
            // Continue with copy to replace potentially corrupted file
        } else if srcChecksum == destChecksum {
            // Duplicate detected
            logger.Log.Infof("⏭️  Skipped duplicate: %s (checksum: %s)",
                filepath.Base(srcPath),
                checksum.FormatChecksum(srcChecksum))
            c.tracker.Register(srcChecksum, destPath)
            c.stats.Skipped++
            return nil
        }
    }

    // 3. Check if this checksum was already copied in this session
    if c.tracker.IsDuplicate(srcChecksum) {
        originalPath, _ := c.tracker.GetOriginalPath(srcChecksum)
        logger.Log.Infof("⏭️  Skipped duplicate: %s (same as %s, checksum: %s)",
            filepath.Base(srcPath),
            filepath.Base(originalPath),
            checksum.FormatChecksum(srcChecksum))
        c.stats.Skipped++
        return nil
    }

    // 4. Perform copy
    tmpPath := destPath + ".tmp"
    if err := copyFileData(srcPath, tmpPath); err != nil {
        return fmt.Errorf("copy failed: %w", err)
    }

    // 5. Verify copied file
    copiedChecksum, err := c.hasher.Calculate(tmpPath)
    if err != nil {
        os.Remove(tmpPath)
        return fmt.Errorf("failed to verify copied file: %w", err)
    }

    if copiedChecksum != srcChecksum {
        os.Remove(tmpPath)
        return fmt.Errorf("checksum mismatch: source=%s, copied=%s",
            checksum.FormatChecksum(srcChecksum),
            checksum.FormatChecksum(copiedChecksum))
    }

    // 6. Atomic rename
    if err := os.Rename(tmpPath, destPath); err != nil {
        os.Remove(tmpPath)
        return fmt.Errorf("failed to finalize copy: %w", err)
    }

    // 7. Register in tracker
    c.tracker.Register(srcChecksum, destPath)
    c.stats.Copied++

    logger.Log.Infof("✅ Copied: %s → %s (checksum: %s)",
        filepath.Base(srcPath),
        destPath,
        checksum.FormatChecksum(srcChecksum))

    return nil
}

func copyFileData(src, dst string) error {
    sourceFile, err := os.Open(src)
    if err != nil {
        return err
    }
    defer sourceFile.Close()

    destFile, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer destFile.Close()

    // Use 1MB buffer for large files
    buf := make([]byte, 1024*1024)
    _, err = io.CopyBuffer(destFile, sourceFile, buf)
    return err
}
```

### 2.4 Modifications des Fichiers Existants

#### 2.4.1 Configuration (`internal/config/config.go`)

```go
// Add new fields to Config struct
type Config struct {
    // ... existing fields ...

    // Copy-only mode
    CopyOnly      bool   `mapstructure:"copy_only"`
    VerifyChecksum bool  `mapstructure:"verify_checksum"`  // Always true for copy-only
}

// Add validation
func (c *Config) Validate() error {
    // ... existing validation ...

    // Check incompatible flags
    if c.CopyOnly {
        if c.Quality != 0 {
            return fmt.Errorf("--quality cannot be used with --copy-only mode")
        }
        if c.Format != "" {
            return fmt.Errorf("--format cannot be used with --copy-only mode")
        }
        if c.VideoCodec != "" {
            return fmt.Errorf("--video-codec cannot be used with --copy-only mode")
        }
        if c.VideoQuality != 0 {
            return fmt.Errorf("--video-quality cannot be used with --copy-only mode")
        }

        // Force checksum verification in copy-only mode
        c.VerifyChecksum = true
    }

    return nil
}
```

#### 2.4.2 CLI (`cmd/root.go`)

```go
// Add new flags
rootCmd.PersistentFlags().Bool("copy-only", false,
    "Copy files without conversion (archive mode)")
rootCmd.PersistentFlags().Bool("verify-checksum", false,
    "Verify file integrity using checksums (automatic in copy-only mode)")

// Bind to viper
viper.BindPFlag("copy_only", rootCmd.PersistentFlags().Lookup("copy-only"))
viper.BindPFlag("verify_checksum", rootCmd.PersistentFlags().Lookup("verify-checksum"))
```

#### 2.4.3 Converter (`internal/converter/converter.go`)

```go
// Add new fields
type Converter struct {
    // ... existing fields ...

    hasher  checksum.Hasher
    tracker *checksum.Tracker
}

// Update NewConverter
func NewConverter(cfg *config.Config, log *logger.Logger) *Converter {
    return &Converter{
        // ... existing initialization ...
        hasher:  checksum.NewXXHash64Hasher(),
        tracker: checksum.NewTracker(),
    }
}

// Update Convert method
func (c *Converter) Convert() error {
    // ... existing initialization ...

    // Choose processing mode
    if c.config.CopyOnly {
        logger.Log.Info("🗂️  Running in COPY-ONLY mode (archive)")
        return c.runCopyMode()
    } else {
        logger.Log.Info("🔄 Running in CONVERSION mode (optimize)")
        return c.runConversionMode()
    }
}

// New method
func (c *Converter) runCopyMode() error {
    // Similar to current Convert() but:
    // 1. Skip conversion logic
    // 2. Use CopyFile instead of convertImage/convertVideo
    // 3. Process all file types uniformly
    // 4. Show duplicate statistics at end

    // ... implementation ...

    // Final stats
    unique, duplicates := c.tracker.Stats()
    logger.Log.Infof("📊 Checksum Stats: %d unique files, %d duplicates skipped",
        unique, duplicates)
}
```

#### 2.4.4 Statistics (`internal/converter/stats.go`)

```go
// Add new fields
type Stats struct {
    // ... existing fields ...

    Copied       int
    Duplicates   int
    ChecksumTime time.Duration
}

// Update PrintSummary
func (s *Stats) PrintSummary() {
    // ... existing output ...

    if s.Copied > 0 {
        fmt.Printf("  Copied:    %d files\n", s.Copied)
    }
    if s.Duplicates > 0 {
        fmt.Printf("  Duplicates: %d files (skipped)\n", s.Duplicates)
    }
    if s.ChecksumTime > 0 {
        fmt.Printf("  Checksum:  %s\n", s.ChecksumTime.Round(time.Second))
    }
}
```

---

## 3. Nouvelle Structure du README

### 3.1 Plan du README

```markdown
# MediaFlow

> Organisateur et archiveur de médias avec conversion optionnelle

## Vue d'ensemble

MediaFlow est un outil professionnel pour gérer vos workflows photo et vidéo :
- **Mode Archive** : Sauvegarde intelligente sur NAS avec détection de doublons
- **Mode Conversion** : Optimisation vers formats modernes (AVIF, WebP, H.265, AV1)

## Cas d'usage

### 1. Archivage sécurisé (NAS)
Copie vos cartes SD sur NAS avec :
- Organisation automatique par date (YYYY/MM-Mois/YYYY-MM-DD)
- Vérification d'intégrité (checksum xxHash64)
- Détection de doublons
- Conservation de la qualité originale

### 2. Conversion pour web/partage
Optimise vos médias :
- Images : AVIF/WebP (économie 50-80%)
- Vidéos : H.265/AV1 (économie 30-50%)
- Traitement parallèle
- Estimation coûts AWS S3

### 3. Workflow complet
Carte SD → NAS (archive) → Local (optimisé) → Web/Cloud

## Installation

[...]

## Guide de démarrage rapide

### Mode Archive (--copy-only)
bash
# Backup carte SD vers NAS
mediaflow --copy-only /Volumes/SD-Card /Volumes/NAS/Photos-Master

# Dry-run pour tester
mediaflow --copy-only --dry-run /source /dest


### Mode Conversion (défaut)
bash
# Convertir depuis NAS vers disque local
mediaflow /Volumes/NAS/Photos-Master/2024 ~/Photos-Optimized

# Options de qualité
mediaflow --quality 85 --format avif /source /dest


## Exemples d'utilisation

### Workflow photographe
bash
# 1. Backup immédiat après shooting
mediaflow --copy-only /Volumes/SD-Card /Volumes/NAS/Archive

# 2. Conversion pour client (quelques jours plus tard)
mediaflow --quality 90 /Volumes/NAS/Archive/2024/11-November /tmp/client-preview

# 3. Web gallery (basse qualité)
mediaflow --quality 75 --format webp /Volumes/NAS/Archive/2024 ~/website/gallery


### Workflow vidéaste
bash
# 1. Archive rushes 4K sur NAS
mediaflow --copy-only /Volumes/RED-Card /Volumes/NAS/Rushes-2024

# 2. Proxies pour montage
mediaflow --video-codec h264 --video-quality 23 /Volumes/NAS/Rushes-2024/11-November ~/Proxies


## Architecture

[Diagramme du workflow]

## Documentation complète

- [Configuration](docs/configuration.md)
- [Formats supportés](docs/formats.md)
- [Performance et optimisation](docs/performance.md)
- [API et intégration](docs/api.md)

## Sécurité et fiabilité

- ✅ Idempotent (peut être relancé sans risque)
- ✅ Vérification d'intégrité (checksums)
- ✅ Détection de corruption
- ✅ Récupération automatique après crash
- ✅ Conservation des originaux par défaut
- ✅ Atomicité des opérations

## Performance

- Traitement parallèle (worker pools)
- Checksum haute vitesse (xxHash64, ~10 GB/s)
- Optimisé pour SSD et NAS
- Support macOS, Linux, Windows

## Licence

MIT

## Contribuer

[...]
```

---

## 4. Plan d'Implémentation par Phases

### Phase 1: Infrastructure Checksum (1-2 jours)

**Objectifs:**
- Implémenter le système de checksum xxHash64
- Créer le tracker de doublons
- Tests unitaires

**Livrables:**
- [ ] `internal/checksum/checksum.go` avec xxHash64
- [ ] `internal/checksum/tracker.go` avec gestion thread-safe
- [ ] `internal/checksum/checksum_test.go` avec benchmarks
- [ ] Documentation godoc

**Tests:**
```bash
go test -v ./internal/checksum/...
go test -bench=. ./internal/checksum/...
```

---

### Phase 2: Mode Copy-Only (2-3 jours)

**Objectifs:**
- Implémenter le flag `--copy-only`
- Validation des flags incompatibles
- Logique de copie avec vérification

**Livrables:**
- [ ] Modification `internal/config/config.go`
- [ ] Modification `cmd/root.go`
- [ ] Nouveau fichier `internal/converter/copy_mode.go`
- [ ] Update `internal/converter/converter.go`

**Tests:**
```bash
# Test copie simple
./mediaflow --copy-only --dry-run /source /dest

# Test erreurs flags incompatibles
./mediaflow --copy-only --quality 85 /source /dest  # doit échouer

# Test copie réelle
./mediaflow --copy-only /tmp/test-source /tmp/test-dest
```

---

### Phase 3: Détection de Doublons (1-2 jours)

**Objectifs:**
- Intégrer le tracker dans le workflow de copie
- Logging des doublons détectés
- Statistiques finales

**Livrables:**
- [ ] Intégration tracker dans `copy_mode.go`
- [ ] Amélioration des logs
- [ ] Statistiques dans `internal/converter/stats.go`

**Tests:**
```bash
# Créer doublons de test
cp /tmp/source/image1.jpg /tmp/source/image1_copy.jpg

# Vérifier détection
./mediaflow --copy-only /tmp/source /tmp/dest
# Doit afficher "Skipped duplicate: image1_copy.jpg"
```

---

### Phase 4: Documentation et Tests (2-3 jours)

**Objectifs:**
- Réécrire le README complet
- Mettre à jour CLAUDE.md
- Tests end-to-end
- Benchmarks de performance

**Livrables:**
- [ ] Nouveau README.md avec exemples
- [ ] Update CLAUDE.md (architecture)
- [ ] Tests d'intégration
- [ ] Benchmarks (vitesse de checksum sur gros fichiers)
- [ ] Documentation utilisateur

**Tests:**
```bash
# Test workflow complet
./test_full_workflow.sh

# Benchmarks
./benchmark_checksum.sh

# Test avec vraie carte SD (si disponible)
./mediaflow --copy-only /Volumes/SD-Card /tmp/test-backup
```

---

### Phase 5: Renommage et Release (1 jour)

**Objectifs:**
- Renommer le projet vers nouveau nom choisi
- Mise à jour de tous les chemins/imports
- Tag version 2.0.0
- Communication

**Livrables:**
- [ ] Renommage repo (si applicable)
- [ ] Update go.mod avec nouveau nom
- [ ] Update imports dans tous les fichiers
- [ ] Tag git v2.0.0
- [ ] Changelog complet
- [ ] Release notes

---

## 5. Considérations Techniques Supplémentaires

### 5.1 Performance

**Benchmarks attendus (xxHash64):**
- Fichier RAW 50MB : ~5ms
- Vidéo 4K 5GB : ~500ms
- Carte SD 64GB : ~6-7 secondes de checksum total

**Optimisations:**
- Buffer de 1MB pour io.Copy
- Buffer de 32KB pour checksum
- Parallélisation des copies (workers)
- Checksums calculés pendant la copie (stream)

### 5.2 Logging Amélioré

**Nouveaux types de logs:**
```
🗂️  COPY MODE: Starting archive to /Volumes/NAS
📂 Scanning source: /Volumes/SD-Card
📊 Found 1,234 files (photos: 890, videos: 344)

✅ Copied: IMG_1234.CR2 → 2024/11-November/2024-11-01/images/ (checksum: 3f8a9b2c4d5e6f7a)
⏭️  Skipped duplicate: IMG_1234.JPG (same as IMG_1234.CR2, checksum: 3f8a9b2c4d5e6f7a)
⚠️  Warning: Failed to extract date for IMG_9999.JPG, using file mtime

📊 Summary:
   Copied:     1,180 files
   Duplicates: 54 files (skipped)
   Errors:     0
   Time:       2m34s
   Checksum:   7.2s (4.7% overhead)
```

### 5.3 Gestion d'Erreurs

**Cas d'erreur critiques:**
1. Checksum source échoue → skip file + log error
2. Copie échoue → retry 1x, puis skip + log
3. Checksum destination ne matche pas → delete + retry 1x
4. Espace disque insuffisant → stop + log clear message
5. Permission denied → skip + log

**Récupération:**
- Toujours nettoyer les `.tmp` en cas d'échec
- Logger tous les échecs dans `conversion.log`
- Statistiques d'erreurs à la fin

### 5.4 Tests à Écrire

```go
// Test checksum consistency
func TestChecksumConsistency(t *testing.T)

// Test duplicate detection
func TestDuplicateDetection(t *testing.T)

// Test copy integrity
func TestCopyIntegrity(t *testing.T)

// Test concurrent access to tracker
func TestTrackerConcurrency(t *testing.T)

// Test incompatible flags
func TestIncompatibleFlags(t *testing.T)

// Benchmark checksum speed
func BenchmarkXXHash64(b *testing.B)
```

---

## 6. Dépendances Nouvelles

### 6.1 Go Modules à Ajouter

```bash
go get github.com/cespare/xxhash/v2
```

### 6.2 Mise à Jour go.mod

```go
module github.com/yourusername/mediaflow

go 1.21

require (
    github.com/cespare/xxhash/v2 v2.2.0
    // ... existing dependencies ...
)
```

---

## 7. Migration pour Utilisateurs Existants

### 7.1 Rétrocompatibilité

**Comportement par défaut:** Mode conversion (actuel) reste inchangé
- Commande `./media-converter /source /dest` fonctionne comme avant
- Aucun breaking change pour utilisateurs existants

**Nouveaux utilisateurs:** Documentation claire des deux modes

### 7.2 Guide de Migration

```markdown
## Migration vers MediaFlow 2.0

### Changement de nom
- Ancien : `media-converter`
- Nouveau : `mediaflow`

### Nouveaux workflows disponibles

#### Ancien workflow (toujours supporté)
bash
./media-converter /source /dest


#### Nouveau workflow recommandé
bash
# 1. Archive d'abord
mediaflow --copy-only /source /nas

# 2. Convertir ensuite
mediaflow /nas /dest


### Pas de breaking changes
Tous vos scripts existants continuent de fonctionner.
```

---

## 8. Checklist Finale

### Avant Release v2.0.0

**Code:**
- [ ] Tous les tests passent
- [ ] Benchmarks validés
- [ ] Pas de regression sur mode conversion
- [ ] Gestion d'erreurs robuste
- [ ] Logs clairs et utiles

**Documentation:**
- [ ] README complet et clair
- [ ] CLAUDE.md à jour
- [ ] Exemples testés
- [ ] Changelog détaillé

**Tests:**
- [ ] Test sur macOS
- [ ] Test sur Linux (si possible)
- [ ] Test avec vraie carte SD
- [ ] Test avec gros volumes (100GB+)
- [ ] Test détection doublons
- [ ] Test récupération après crash

**Polish:**
- [ ] Choix final du nom
- [ ] Messages d'erreur clairs
- [ ] Progress bars fluides
- [ ] Statistiques utiles

---

## 9. Questions Ouvertes

### 9.1 À Décider

1. **Nom final du projet:** MediaFlow vs alternatives ?
2. **Namespace Go:** `github.com/username/mediaflow` ?
3. **Logo/branding:** Nécessaire ?
4. **Release strategy:** GitHub releases ? Homebrew ? Go install ?

### 9.2 Améliorations Futures (v2.1+)

- [ ] Mode `--verify` pour checker intégrité d'un NAS existant
- [ ] Export checksums vers fichier (`.checksums.json`)
- [ ] Interface web pour monitoring
- [ ] Support cloud (S3, Google Drive, etc.)
- [ ] Détection de doublons fuzzy (images similaires)
- [ ] Géolocalisation et organisation par lieu

---

## 10. Conclusion

Ce plan transforme le projet d'un simple convertisseur vers un **gestionnaire complet de workflow média**. L'implémentation est progressive et sans breaking changes, permettant une adoption en douceur.

**Effort estimé:** 8-12 jours de développement
**Complexité:** Moyenne (réutilisation de 70% du code existant)
**Impact:** Majeur (nouveau positionnement, nouveaux cas d'usage)

**Next steps:**
1. Valider le choix du nom final
2. Commencer Phase 1 (infrastructure checksum)
3. Tests itératifs après chaque phase

---

**Questions ? Feedback ?**
Ouvre une issue ou contacte-moi directement !
