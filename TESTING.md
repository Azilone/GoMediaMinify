# Guide de Tests - GoMediaMinify

Ce document explique comment tester le programme GoMediaMinify automatiquement.

## 🎯 Objectif

Assurer que chaque nouvelle fonctionnalité fonctionne correctement et que les modifications n'introduisent pas de régressions.

## 📦 Prérequis

Pour exécuter les tests, vous devez avoir installé :

- **Go** (version 1.21 ou supérieure)
- **FFmpeg** (pour les tests de vidéos)
- **ImageMagick** (pour les tests d'images)

### Installation des dépendances

**macOS:**
```bash
brew install ffmpeg imagemagick
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt-get update
sudo apt-get install -y ffmpeg imagemagick
```

**Windows:**
- Télécharger [FFmpeg](https://ffmpeg.org/download.html)
- Télécharger [ImageMagick](https://imagemagick.org/script/download.php#windows)

## 🧪 Types de Tests

### 1. Tests Unitaires
Tests des fonctions individuelles et de la logique métier.

**Localisation:**
- `internal/utils/utils_test.go` - Tests des fonctions utilitaires
- `internal/converter/adaptive_limiter_test.go` - Tests du limiteur adaptatif
- `internal/converter/adaptive_workers_test.go` - Tests des workers adaptatifs

### 2. Tests d'Intégration
Tests du workflow complet de conversion.

**Localisation:**
- `internal/converter/converter_test.go` - Tests de conversion d'images
- `internal/converter/video_test.go` - Tests de conversion de vidéos

## 🚀 Exécuter les Tests

### Méthode 1: Utiliser Make (Recommandé)

```bash
# Exécuter tous les tests
make test

# Exécuter les tests avec couverture
make test-coverage

# Le fichier coverage.html sera généré pour visualiser la couverture
```

### Méthode 2: Utiliser la commande Go directement

```bash
# Tous les tests
go test -v ./...

# Tests avec couverture
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Tests d'un package spécifique
go test -v ./internal/converter/
go test -v ./internal/utils/
```

### Méthode 3: Tests individuels

```bash
# Exécuter un test spécifique
go test -v -run TestImageConversion ./internal/converter/

# Exécuter tous les tests de vidéos
go test -v -run TestVideo ./internal/converter/
```

## 📊 Tests Disponibles

### Tests de Conversion d'Images

| Test | Description |
|------|-------------|
| `TestImageConversion` | Teste la conversion complète d'une image (JPG → AVIF) |
| `TestImageConversionWithDelete` | Teste la conversion avec suppression de l'original |
| `TestDryRunMode` | Vérifie que le mode dry-run ne convertit pas réellement |
| `TestIdempotency` | Vérifie qu'une conversion déjà faite est ignorée |
| `TestFindFiles` | Teste la découverte de fichiers dans les dossiers |

### Tests de Conversion de Vidéos

| Test | Description |
|------|-------------|
| `TestVideoConversion` | Teste la conversion complète d'une vidéo |
| `TestVideoConversionH265` | Teste la conversion en H.265 |
| `TestVideoDryRun` | Vérifie le mode dry-run pour les vidéos |
| `TestVideoIdempotency` | Vérifie l'idempotence des conversions vidéo |

### Tests Utilitaires

| Test | Description |
|------|-------------|
| `TestHasExtension` | Vérifie la détection d'extensions de fichiers |
| `TestCleanFilename` | Teste le nettoyage de noms de fichiers |
| `TestGetMonthName` | Teste la localisation des noms de mois |
| `TestShouldSkipSystemEntry` | Teste le filtrage de fichiers système |
| `TestCreateDestinationPath` | Teste la création de chemins de destination |
| `TestIsValidDate` | Teste la validation de dates |

## 🤖 Tests Automatiques (CI/CD)

### Configuration GitHub Actions

Pour activer les tests automatiques via GitHub Actions, un fichier de workflow exemple est fourni dans `test.yml.example`.

**Option 1 : Via l'interface GitHub (Recommandé)**
1. Allez sur GitHub.com et ouvrez votre repository
2. Cliquez sur **Actions** > **New workflow** > **set up a workflow yourself**
3. Copiez le contenu de `test.yml.example` dans l'éditeur
4. Nommez le fichier `test.yml`
5. Commit le fichier

**Option 2 : En local**
1. Créez le fichier `.github/workflows/test.yml` avec le contenu de `test.yml.example`
2. Commit et push depuis votre machine locale

Une fois activé, les tests s'exécutent automatiquement à chaque :
- Push sur la branche `main`
- Push sur une branche `claude/*`
- Création d'une Pull Request

### Voir les résultats

1. Aller sur l'onglet **Actions** du repository GitHub
2. Sélectionner le workflow **Tests**
3. Voir les résultats pour chaque plateforme (Ubuntu, macOS)

### Badge de statut

Vous pouvez ajouter un badge dans le README pour afficher le statut des tests :

```markdown
![Tests](https://github.com/Azilone/GoMediaMinify/workflows/Tests/badge.svg)
```

## 🔍 Debugger un Test qui Échoue

### 1. Exécuter avec plus de détails

```bash
go test -v -run TestNomDuTest ./internal/converter/
```

### 2. Voir les logs de conversion

Les tests créent des fichiers temporaires dans `/tmp/`. Les logs de conversion sont disponibles dans ces dossiers temporaires.

### 3. Désactiver le nettoyage automatique

Modifiez temporairement le test pour ne pas nettoyer les fichiers :

```go
// Commentez cette ligne pour garder les fichiers de test
// defer os.RemoveAll(tempDir)
```

## 💡 Ajouter de Nouveaux Tests

### Exemple : Tester un nouveau format d'image

```go
func TestImageConversionWebP(t *testing.T) {
    // 1. Créer des dossiers temporaires
    tempDir := t.TempDir()
    sourceDir := filepath.Join(tempDir, "source")
    destDir := filepath.Join(tempDir, "dest")

    // 2. Créer un fichier de test
    testImagePath := filepath.Join(sourceDir, "test.jpg")
    if err := createTestImage(testImagePath); err != nil {
        t.Fatalf("Failed to create test image: %v", err)
    }

    // 3. Configurer le convertisseur
    cfg := &config.Config{
        SourceDir:    sourceDir,
        DestDir:      destDir,
        PhotoFormat:  "webp",  // Nouveau format
        // ... autres options
    }

    // 4. Exécuter la conversion
    converter := NewConverter(cfg, logger)
    err := converter.convertImage(testImagePath)

    // 5. Vérifier les résultats
    if err != nil {
        t.Fatalf("Conversion failed: %v", err)
    }
}
```

## 📈 Couverture de Code

L'objectif est d'avoir **au moins 70% de couverture de code**.

Pour voir la couverture actuelle :

```bash
make test-coverage
# Ouvrir coverage.html dans un navigateur
```

## ⚠️ Tests qui Peuvent Être Ignorés

Certains tests peuvent être ignorés (skippés) si :

- ImageMagick n'est pas installé (tests d'images)
- FFmpeg n'est pas installé (tests de vidéos)
- L'environnement ne supporte pas certains codecs

Ces tests afficheront `SKIP` au lieu de `PASS` ou `FAIL`.

## 🎓 Bonnes Pratiques

1. **Toujours exécuter les tests avant de commit**
   ```bash
   make test
   ```

2. **Ajouter des tests pour chaque nouvelle fonctionnalité**

3. **Utiliser `t.TempDir()` pour les fichiers temporaires**
   - Nettoyage automatique
   - Pas de conflits entre tests

4. **Vérifier la couverture de code régulièrement**
   ```bash
   make test-coverage
   ```

5. **Utiliser des noms de tests descriptifs**
   ```go
   func TestImageConversionWithOrganizeByDate(t *testing.T) { ... }
   ```

## 📞 Besoin d'Aide ?

Si un test échoue :

1. Lisez le message d'erreur complet
2. Vérifiez que toutes les dépendances sont installées
3. Exécutez le test en mode verbose : `go test -v -run TestNom`
4. Ouvrez une issue sur GitHub avec les logs d'erreur

## 🚀 Commandes Rapides

```bash
# Développement quotidien
make test                    # Tous les tests
make test-coverage          # Tests avec couverture
make check                  # Format + vet + tests

# Debug
go test -v ./internal/converter/  # Tests d'un package
go test -run TestImageConversion  # Un test spécifique

# CI/CD local
make clean                  # Nettoyer
make deps                   # Dépendances
make test                   # Tests
make build                  # Construire
```

---

**Note:** Les tests sont votre filet de sécurité. Ils vous permettent de développer rapidement tout en garantissant que rien ne casse ! 🎯
