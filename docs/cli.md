# Documentation CLI Magina

## Vue d'Ensemble

Magina est un outil CLI pour la gestion et la migration d'images OCI entre registries.

## Commandes Disponibles

### `magina export`

Exporte des images depuis un registry source vers l'hôte local.

```bash
magina export -c <config-file> [flags]
```

**Flags :**
- `-c, --config` : Fichier de configuration BRMS (obligatoire)
- `-v, --verbose` : Niveau de verbosité (0-3)
- `--clean-on-error` : Nettoie les images en cas d'erreur
- `--resume` : Continue l'opération même en cas d'erreur

**Format BRMS :**
```brms
[protocol://export-host|]
image1:tag1|newimage1:tag1
```

### `magina import`

Importe des images locales vers un registry de destination.

```bash
magina import -c <config-file> [flags]
```

**Flags :**
- `-c, --config` : Fichier de configuration BRMS (obligatoire)
- `-v, --verbose` : Niveau de verbosité (0-3)
- `--clean-on-error` : Nettoie les images en cas d'erreur
- `--resume` : Continue l'opération même en cas d'erreur

**Format BRMS :**
```brms
[|protocol://import-host]
image1:tag1|newimage1:tag1
```

### `magina transfer`

Effectue un transfert complet d'images entre deux registries.

```bash
magina transfer -c <config-file> [flags]
```

**Flags :**
- `-c, --config` : Fichier de configuration BRMS (obligatoire)
- `-v, --verbose` : Niveau de verbosité (0-3)
- `--clean-on-error` : Nettoie les images en cas d'erreur
- `--resume` : Continue l'opération même en cas d'erreur

**Format BRMS :**
```brms
[protocol://source-host|protocol://dest-host]
image1:tag1|newimage1:tag1
```

### `magina validate`

Valide un fichier de configuration BRMS.

```bash
magina validate -c <config-file>
```

**Flags :**
- `-c, --config` : Fichier de configuration BRMS (obligatoire)
- `-v, --verbose` : Niveau de verbosité (0-3)

## Niveaux de Verbosité

- `0` : Silencieux (erreurs uniquement)
- `1` : Normal (succès + erreurs)
- `2` : Détaillé (progression + métadonnées)
- `3` : Debug (toutes les informations)

## Format BRMS Détaillé

### Structure Générale
```brms
[source|destination]
image_mapping1
image_mapping2
!exclusion1
```

### Exemples

#### Export Simple
```brms
[https://registry.company.com|]
app/backend:1.0.0|backend:local
app/frontend:1.0.0|frontend:local
```

#### Import Simple
```brms
[|https://docker.io]
backend:local|company/backend:latest
frontend:local|company/frontend:latest
```

#### Transfert avec Exclusions
```brms
[https://registry.company.com|https://docker.io]
!test/*
!dev/*
app/backend:1.0.0|company/backend:latest
app/frontend:1.0.0|company/frontend:latest
```

## Stockage Local des Images

Magina stocke les images localement sans nécessiter de runtime de conteneur (Docker/Podman). Les images sont stockées sous forme de fichiers dans le système de fichiers local en utilisant le format OCI standard.

### Format de Stockage
Les images exportées sont stockées dans le format OCI natif, ce qui les rend compatibles avec n'importe quel runtime OCI (Docker, Podman, etc.) si vous souhaitez les utiliser plus tard.

### Erreurs Communes
- `failed to parse source image reference` : Format d'image invalide
- `failed to load source image` : Erreur de connexion au registry ou d'authentification
- `failed to save image locally` : Problème d'écriture locale (permissions, espace disque)

Note : Magina ne vérifie pas la présence d'un runtime de conteneur car il n'en a pas besoin pour fonctionner.

## Codes de Retour

- `0` : Succès
- `1` : Erreur générale
- `2` : Erreur de configuration
- `3` : Erreur de connexion
- `4` : Erreur d'authentification

## Variables d'Environnement

```bash
# Proxy
HTTP_PROXY="http://proxy.company.com:3128"
HTTPS_PROXY="http://proxy.company.com:3128"
NO_PROXY="localhost,127.0.0.1"

# Docker
DOCKER_HOST="tcp://localhost:2375"
DOCKER_CERT_PATH="/path/to/certs"
DOCKER_TLS_VERIFY="1"
```

## Exemples d'Utilisation

### Export avec Nettoyage
```bash
magina export -c config.brms --clean-on-error -v 1
```

### Import avec Reprise
```bash
magina import -c config.brms --resume -v 2
```

### Transfert Complet en Mode Debug
```bash
magina transfer -c config.brms --clean-on-error --resume -v 3
```

### Validation Simple
```bash
magina validate -c config.brms
```
