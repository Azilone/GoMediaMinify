# Tauri Desktop Integration Guide

## ✅ JSON Mode Implementation - Complete

Le mode JSON a été implémenté avec succès pour permettre l'intégration avec Tauri (ou toute autre application desktop).

## 🎯 Ce qui a été fait

### 1. Package API JSON (`internal/api/`)

**Fichiers créés :**
- `json_types.go` - Définitions de tous les types d'événements JSON
- `json_writer.go` - Writer pour émettre des événements JSON vers stdout

**Types d'événements supportés :**
- `started` - Début de la conversion
- `progress` - Progression globale
- `file_start` - Début du traitement d'un fichier
- `file_end` - Fin du traitement d'un fichier
- `log` - Messages de log (info/warn/error)
- `error` - Événements d'erreur
- `complete` - Fin de la conversion
- `statistics` - Statistiques détaillées

### 2. Flag CLI `--json-mode`

```bash
./media-converter --json-mode /source /destination
```

Quand activé :
- ✅ Émet des événements JSON sur stdout (une ligne par événement)
- ✅ Les logs traditionnels vont dans `conversion.log` uniquement
- ✅ Pas de sortie colorée ou de barres de progression ANSI
- ✅ Format parsable par machine

### 3. Modifications du code

**Config (`internal/config/config.go`):**
- Ajout du champ `JSONMode bool`
- Support via viper config

**Logger (`internal/logger/logger.go`):**
- Nouveau constructeur `NewLoggerWithMode(path, jsonMode)`
- Méthodes `GetJSONWriter()` et `IsJSONMode()`
- Redirection automatique des logs

**Converter (`internal/converter/converter.go`):**
- Émission d'événements `started` avec la configuration
- Émission d'événements `complete` avec le résultat
- Émission de `statistics` avec les détails de compression

## 📚 Documentation

### Guide complet
Voir [`docs/JSON_MODE.md`](docs/JSON_MODE.md) pour :
- Format détaillé de tous les événements JSON
- Exemples d'intégration (Node.js, Rust, Python)
- Bonnes pratiques
- Troubleshooting

### Exemple pratique
Voir [`examples/json-mode-test.js`](examples/json-mode-test.js) :
```bash
# Tester le mode JSON avec Node.js
node examples/json-mode-test.js ./photos ./converted
```

## 🚀 Prochaines étapes pour Tauri

### Option 1 : Subprocess (Recommandé pour commencer)

**Backend Rust (Tauri):**

```rust
// src-tauri/src/main.rs
use std::process::{Command, Stdio};
use std::io::{BufRead, BufReader};

#[tauri::command]
async fn start_conversion(
    app: tauri::AppHandle,
    source: String,
    dest: String,
) -> Result<(), String> {
    let mut child = Command::new("./resources/media-converter")
        .arg("--json-mode")
        .arg(source)
        .arg(dest)
        .stdout(Stdio::piped())
        .spawn()
        .map_err(|e| e.to_string())?;

    let stdout = child.stdout.take().unwrap();
    let reader = BufReader::new(stdout);

    for line in reader.lines() {
        let line = line.map_err(|e| e.to_string())?;

        if let Ok(event) = serde_json::from_str::<serde_json::Value>(&line) {
            // Émettre vers le frontend
            app.emit_all("conversion-event", &event)
                .map_err(|e| e.to_string())?;
        }
    }

    Ok(())
}
```

**Frontend (React/Vue/Svelte):**

```javascript
import { invoke } from '@tauri-apps/api/tauri';
import { listen } from '@tauri-apps/api/event';

// Écouter les événements
const unlisten = await listen('conversion-event', (event) => {
  const data = event.payload;

  switch(data.type) {
    case 'started':
      console.log(`Starting: ${data.data.total_files} files`);
      break;
    case 'progress':
      updateProgressBar(data.data.progress_percent);
      break;
    case 'complete':
      showCompletionMessage(data.data);
      break;
  }
});

// Démarrer la conversion
await invoke('start_conversion', {
  source: '/path/to/source',
  dest: '/path/to/dest'
});
```

### Option 2 : API HTTP locale (Pour plus tard)

Si vous avez besoin de fonctionnalités avancées :
- Contrôle bidirectionnel (pause/resume/cancel)
- WebSockets pour real-time updates
- Plusieurs conversions simultanées

## 🔧 Compilation et packaging

### Intégrer le binaire Go dans Tauri

**tauri.conf.json:**
```json
{
  "tauri": {
    "bundle": {
      "resources": [
        "resources/media-converter*"
      ]
    }
  }
}
```

**Build script:**
```bash
#!/bin/bash
# build-tauri.sh

# Compiler le binaire Go pour toutes les plateformes
GOOS=darwin GOARCH=amd64 go build -o resources/media-converter-darwin-amd64
GOOS=darwin GOARCH=arm64 go build -o resources/media-converter-darwin-arm64
GOOS=windows GOARCH=amd64 go build -o resources/media-converter-windows-amd64.exe
GOOS=linux GOARCH=amd64 go build -o resources/media-converter-linux-amd64

# Construire l'app Tauri
npm run tauri build
```

## 📝 TODO pour migration complète vers Tauri

- [ ] Créer le projet Tauri
  ```bash
  npm create tauri-app@latest
  ```

- [ ] Créer l'interface utilisateur
  - [ ] File picker pour source/destination
  - [ ] Options de conversion (quality, format, codec)
  - [ ] Progress bars en temps réel
  - [ ] Liste des fichiers traités
  - [ ] Statistiques finales

- [ ] Implémenter les commandes Tauri
  - [ ] `start_conversion` - Lancer la conversion
  - [ ] `cancel_conversion` - Annuler en cours
  - [ ] `get_config` - Récupérer la config actuelle
  - [ ] `save_config` - Sauvegarder les préférences

- [ ] Gérer les dépendances (FFmpeg, ImageMagick)
  - Option 1: Les inclure dans le bundle
  - Option 2: Vérifier leur présence et guider l'installation

- [ ] Packaging multi-plateforme
  - [ ] macOS (.dmg, .app)
  - [ ] Windows (.exe, .msi)
  - [ ] Linux (.AppImage, .deb)

- [ ] Features desktop
  - [ ] Drag & drop de dossiers
  - [ ] Notifications système
  - [ ] Menu bar / System tray
  - [ ] Raccourcis clavier

## 🎨 Suggestions d'UI

```
┌─────────────────────────────────────────────────┐
│  Media Converter                         [_][□][X]
├─────────────────────────────────────────────────┤
│                                                 │
│  Source:      [/path/to/photos  ] [Browse...]  │
│  Destination: [/path/to/output  ] [Browse...]  │
│                                                 │
│  Mode: ◉ Conversion  ○ Copy Only               │
│                                                 │
│  Photo Format:  [AVIF ▼]  Quality: [80 ——●——] │
│  Video Codec:   [H.265▼]  CRF:     [28 ——●——] │
│                                                 │
│  ☑ Keep originals  ☑ Organize by date          │
│  ☑ Hardware acceleration                       │
│                                                 │
│  [         Start Conversion         ]          │
│                                                 │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  47%           │
│  Processing: IMG_1234.jpg (52/110)             │
│  ETA: 2m 15s                                    │
│                                                 │
│  📸 Images: 45/90  🎬 Videos: 7/20             │
│  💾 Saved: 2.3 GB (83% reduction)              │
│                                                 │
└─────────────────────────────────────────────────┘
```

## 🧪 Tester le mode JSON maintenant

```bash
# Build
go build -o media-converter

# Test avec l'exemple Node.js
node examples/json-mode-test.js ./test-source ./test-dest

# Test manuel
./media-converter --json-mode --dry-run ./test-source ./test-dest | jq '.'
```

## 💡 Avantages de cette approche

✅ **Réutilisation du code** - 100% de la logique Go est préservée
✅ **Pas de réécriture** - Seulement ajout du mode JSON
✅ **CLI toujours fonctionnel** - Le CLI standalone continue de marcher
✅ **Migration progressive** - Peut commencer avec subprocess, évoluer vers API
✅ **Débogage facile** - Les deux modes (CLI et JSON) peuvent être testés indépendamment

## 📞 Questions ?

Le mode JSON est maintenant prêt à être utilisé. Pour créer l'application Tauri complète, faites-moi signe et je vous aiderai avec :
- La structure du projet Tauri
- L'interface React/Vue/Svelte
- L'intégration des commandes
- Le packaging multi-plateforme
